package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// PipelineStatus is the JSON response for status queries.
type PipelineStatus struct {
	FileName   string `json:"fileName"`
	Running    bool   `json:"running"`
	Zoomed     bool   `json:"zoomed"`
	ZoomCamera string `json:"zoomCamera,omitempty"`
}

// CameraStatus is the JSON response for camera list queries.
type CameraStatus struct {
	Name     string `json:"name"`
	IDCell   int    `json:"idCell"`
	IPCamera string `json:"ipCamera"`
	Active   bool   `json:"active"`
}

// Pipeline manages a single FFmpeg process that composes a grid of cameras
// into a single RTSP output stream, published to MediaMTX.
type Pipeline struct {
	mu           sync.RWMutex
	config       PontonScreenConfiguration
	ffmpegPath   string
	mediamtxRTSP string
	rtspPort     string

	ctx    context.Context
	cancel context.CancelFunc

	running    bool
	zoomed     bool
	zoomCamera string

	cmd       *exec.Cmd
	restartCh chan struct{}
}

func NewPipeline(config PontonScreenConfiguration, cfg *Config) *Pipeline {
	return &Pipeline{
		config:       config,
		ffmpegPath:   cfg.FFmpegPath,
		mediamtxRTSP: cfg.MediaMTXRTSP,
		rtspPort:     cfg.RTSPPort,
		restartCh:    make(chan struct{}, 1),
	}
}

// Start launches the pipeline's run loop in a goroutine.
func (p *Pipeline) Start() {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.running = true
	p.mu.Unlock()

	go p.runLoop()
}

// Stop terminates the pipeline and its FFmpeg process.
func (p *Pipeline) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	p.cancel()
	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}
	p.mu.Unlock()
}

// UpdateConfig replaces the pipeline's configuration and triggers a restart.
func (p *Pipeline) UpdateConfig(config PontonScreenConfiguration) {
	p.mu.Lock()
	p.config = config
	p.mu.Unlock()
	p.triggerRestart()
}

// Zoom switches the pipeline to show a single camera at full resolution.
func (p *Pipeline) Zoom(cameraName string) error {
	p.mu.Lock()
	found := false
	for _, cell := range p.config.GridConfiguration.GridCells {
		if cell.On && cell.Name == cameraName {
			found = true
			break
		}
	}
	if !found {
		p.mu.Unlock()
		return fmt.Errorf("camera %q not found or inactive", cameraName)
	}
	p.zoomed = true
	p.zoomCamera = cameraName
	p.mu.Unlock()

	p.triggerRestart()
	return nil
}

// Unzoom restores the grid composition view.
func (p *Pipeline) Unzoom() {
	p.mu.Lock()
	p.zoomed = false
	p.zoomCamera = ""
	p.mu.Unlock()

	p.triggerRestart()
}

// Status returns the current pipeline state.
func (p *Pipeline) Status() PipelineStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return PipelineStatus{
		FileName:   p.config.FileName,
		Running:    p.running,
		Zoomed:     p.zoomed,
		ZoomCamera: p.zoomCamera,
	}
}

// Cameras returns the list of active cameras in the grid.
func (p *Pipeline) Cameras() []CameraStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var cameras []CameraStatus
	for _, cell := range p.config.GridConfiguration.GridCells {
		if cell.On {
			cameras = append(cameras, CameraStatus{
				Name:     cell.Name,
				IDCell:   cell.IDCell,
				IPCamera: cell.IPCamera,
				Active:   true,
			})
		}
	}
	return cameras
}

// triggerRestart kills the current FFmpeg process and signals immediate restart.
func (p *Pipeline) triggerRestart() {
	p.mu.RLock()
	cmd := p.cmd
	p.mu.RUnlock()

	if cmd != nil && cmd.Process != nil {
		cmd.Process.Kill()
	}

	select {
	case p.restartCh <- struct{}{}:
	default:
	}
}

// runLoop is the main loop that keeps FFmpeg running with automatic restart.
func (p *Pipeline) runLoop() {
	fileName := p.config.FileName
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		args := p.buildArgs()
		if args == nil {
			log.Printf("[pipeline:%s] no active cameras, waiting...", fileName)
			select {
			case <-p.ctx.Done():
				return
			case <-p.restartCh:
				backoff = time.Second
				continue
			case <-time.After(5 * time.Second):
				continue
			}
		}

		log.Printf("[pipeline:%s] starting FFmpeg", fileName)

		cmd := exec.CommandContext(p.ctx, p.ffmpegPath, args...)
		cmd.Stdout = &logWriter{prefix: fileName}
		cmd.Stderr = &logWriter{prefix: fileName}

		p.mu.Lock()
		p.cmd = cmd
		p.mu.Unlock()

		err := cmd.Run()

		p.mu.Lock()
		p.cmd = nil
		p.mu.Unlock()

		select {
		case <-p.ctx.Done():
			return
		default:
		}

		if err != nil {
			log.Printf("[pipeline:%s] FFmpeg exited: %v", fileName, err)
		}

		// Wait for restart signal (immediate) or backoff timer
		select {
		case <-p.ctx.Done():
			return
		case <-p.restartCh:
			backoff = time.Second
			log.Printf("[pipeline:%s] restarting (triggered)", fileName)
		case <-time.After(backoff):
			log.Printf("[pipeline:%s] restarting after %v", fileName, backoff)
			if backoff < maxBackoff {
				backoff *= 2
			}
		}
	}
}

// buildArgs returns the FFmpeg arguments for the current mode (grid or zoom).
func (p *Pipeline) buildArgs() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.zoomed {
		return p.buildZoomArgs()
	}
	return p.buildGridArgs()
}

// buildRTSPUrl constructs the RTSP source URL for a camera cell based on selectFlow.
//
// selectFlow values:
//   1 = NVR primary stream
//   2 = NVR secondary stream
//   3 = MediaMTX camera1
//   4 = MediaMTX camera2
//   5 = Direct camera primary
//   6 = Direct camera secondary
func (p *Pipeline) buildRTSPUrl(cell PontonGridCell, selectFlow int) string {
	switch selectFlow {
	case 1: // NVR primary
		u := &url.URL{
			Scheme: "rtsp",
			User:   url.UserPassword(cell.User, cell.Pass),
			Host:   fmt.Sprintf("%s:%s", cell.IPNVR, p.rtspPort),
			Path:   fmt.Sprintf("/Streaming/Channels/%d01", cell.NVRChannel),
		}
		return u.String()
	case 2: // NVR secondary
		u := &url.URL{
			Scheme: "rtsp",
			User:   url.UserPassword(cell.User, cell.Pass),
			Host:   fmt.Sprintf("%s:%s", cell.IPNVR, p.rtspPort),
			Path:   fmt.Sprintf("/Streaming/Channels/%d02", cell.NVRChannel),
		}
		return u.String()
	case 3: // MediaMTX camera1
		return fmt.Sprintf("%s/%s", p.mediamtxRTSP, cell.MediaMTXCamera1)
	case 4: // MediaMTX camera2
		return fmt.Sprintf("%s/%s", p.mediamtxRTSP, cell.MediaMTXCamera2)
	case 5: // Direct camera primary
		u := &url.URL{
			Scheme: "rtsp",
			User:   url.UserPassword(cell.User, cell.Pass),
			Host:   fmt.Sprintf("%s:%s", cell.IPCamera, p.rtspPort),
			Path:   "/Streaming/Channels/101",
		}
		return u.String()
	case 6: // Direct camera secondary
		u := &url.URL{
			Scheme: "rtsp",
			User:   url.UserPassword(cell.User, cell.Pass),
			Host:   fmt.Sprintf("%s:%s", cell.IPCamera, p.rtspPort),
			Path:   "/Streaming/Channels/102",
		}
		return u.String()
	default:
		return p.buildRTSPUrl(cell, 1)
	}
}

// encoderArgs returns FFmpeg encoder flags based on hardwareEncoding mode.
//   1 = CPU only (libx264)
//   2,3,4 = GPU encoding (h264_nvenc)
func (p *Pipeline) encoderArgs() []string {
	switch p.config.HardwareEncoding {
	case 2, 3, 4:
		return []string{"-c:v", "h264_nvenc", "-preset", "p4", "-tune", "ll", "-rc", "cbr"}
	default:
		return []string{"-c:v", "libx264", "-preset", "fast", "-tune", "zerolatency"}
	}
}

// buildGridArgs constructs FFmpeg arguments for the grid composition mode.
// It pulls N RTSP sources, scales each to a cell, and composes them with xstack.
func (p *Pipeline) buildGridArgs() []string {
	gc := p.config.GridConfiguration
	rows := gc.Rows
	cols := gc.Columns
	totalCells := rows * cols

	if totalCells == 0 {
		return nil
	}

	width := gc.WidthResolution
	height := gc.HeightResolution
	if width == 0 {
		width = 1920
	}
	if height == 0 {
		height = 1080
	}
	cellW := width / cols
	cellH := height / rows
	fps := gc.FPS
	if fps == 0 {
		fps = 20
	}
	gop := gc.GOP
	if gop == 0 {
		gop = 25
	}
	bitrate := p.config.Bitrate
	if bitrate == 0 {
		bitrate = 3000
	}

	args := []string{"-y", "-loglevel", "warning"}

	// Add RTSP inputs for active cells only
	inputIdx := 0
	inputMap := make(map[int]int) // cell index -> ffmpeg input index

	for i := 0; i < totalCells && i < len(gc.GridCells); i++ {
		cell := gc.GridCells[i]
		if cell.On {
			rtspUrl := p.buildRTSPUrl(cell, gc.SelectFlow)
			args = append(args,
				"-rtsp_transport", "tcp",
				"-timeout", "5000000",
				"-thread_queue_size", "512",
				"-i", rtspUrl,
			)
			inputMap[i] = inputIdx
			inputIdx++
		}
	}

	if inputIdx == 0 {
		return nil
	}

	// Build filter_complex: scale each input + generate black for empty cells + xstack
	var filterParts []string
	var stackLabels []string
	var layoutParts []string

	for i := 0; i < totalCells; i++ {
		row := i / cols
		col := i % cols
		label := fmt.Sprintf("c%d", i)

		if idx, ok := inputMap[i]; ok {
			// Active cell: scale camera feed to cell dimensions
			wFactor := 1.0
			if i < len(gc.GridCells) && gc.GridCells[i].WFactor > 0 {
				wFactor = gc.GridCells[i].WFactor
			}
			scaledW := int(float64(cellW) * wFactor)

			filterParts = append(filterParts,
				fmt.Sprintf("[%d:v]setpts=PTS-STARTPTS,scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black[%s]",
					idx, scaledW, cellH, cellW, cellH, label),
			)
		} else {
			// Inactive cell: generate solid black
			filterParts = append(filterParts,
				fmt.Sprintf("color=black:s=%dx%d:r=%d[%s]", cellW, cellH, fps, label),
			)
		}

		stackLabels = append(stackLabels, fmt.Sprintf("[%s]", label))
		layoutParts = append(layoutParts, fmt.Sprintf("%d_%d", col*cellW, row*cellH))
	}

	// Compose all cells into a single frame with xstack
	filterComplex := strings.Join(filterParts, ";") + ";" +
		strings.Join(stackLabels, "") +
		fmt.Sprintf("xstack=inputs=%d:layout=%s[out]", totalCells, strings.Join(layoutParts, "|"))

	args = append(args, "-filter_complex", filterComplex)
	args = append(args, "-map", "[out]", "-an")

	// Encoder
	args = append(args, p.encoderArgs()...)

	// Output parameters
	args = append(args,
		"-b:v", fmt.Sprintf("%dk", bitrate),
		"-maxrate", fmt.Sprintf("%dk", bitrate),
		"-bufsize", fmt.Sprintf("%dk", bitrate*2),
		"-r", fmt.Sprintf("%d", fps),
		"-g", fmt.Sprintf("%d", gop),
		"-f", "rtsp",
		"-rtsp_transport", "tcp",
		fmt.Sprintf("%s/%s", p.mediamtxRTSP, p.config.FileName),
	)

	return args
}

// buildZoomArgs constructs FFmpeg arguments for single-camera zoom mode.
// It pulls one RTSP source and scales it to full output resolution.
func (p *Pipeline) buildZoomArgs() []string {
	gc := p.config.GridConfiguration

	// Find the target camera by name
	var target PontonGridCell
	found := false
	for _, cell := range gc.GridCells {
		if cell.On && cell.Name == p.zoomCamera {
			target = cell
			found = true
			break
		}
	}
	if !found {
		return nil
	}

	width := gc.WidthResolution
	height := gc.HeightResolution
	if width == 0 {
		width = 1920
	}
	if height == 0 {
		height = 1080
	}
	fps := gc.FPS
	if fps == 0 {
		fps = 20
	}
	gop := gc.GOP
	if gop == 0 {
		gop = 25
	}
	bitrate := p.config.Bitrate
	if bitrate == 0 {
		bitrate = 3000
	}

	rtspUrl := p.buildRTSPUrl(target, gc.SelectFlow)

	args := []string{
		"-y", "-loglevel", "warning",
		"-rtsp_transport", "tcp",
		"-timeout", "5000000",
		"-i", rtspUrl,
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black",
			width, height, width, height),
		"-an",
	}

	// Encoder
	args = append(args, p.encoderArgs()...)

	// Output
	args = append(args,
		"-b:v", fmt.Sprintf("%dk", bitrate),
		"-maxrate", fmt.Sprintf("%dk", bitrate),
		"-bufsize", fmt.Sprintf("%dk", bitrate*2),
		"-r", fmt.Sprintf("%d", fps),
		"-g", fmt.Sprintf("%d", gop),
		"-f", "rtsp",
		"-rtsp_transport", "tcp",
		fmt.Sprintf("%s/%s", p.mediamtxRTSP, p.config.FileName),
	)

	return args
}

// logWriter prefixes each line of FFmpeg output with the pipeline name.
type logWriter struct {
	prefix string
}

func (w *logWriter) Write(p []byte) (int, error) {
	lines := strings.Split(strings.TrimRight(string(p), "\n"), "\n")
	for _, line := range lines {
		if line != "" {
			log.Printf("[ffmpeg:%s] %s", w.prefix, line)
		}
	}
	return len(p), nil
}
