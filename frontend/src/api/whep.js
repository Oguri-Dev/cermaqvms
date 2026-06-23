/**
 * WHEP (WebRTC-HTTP Egress Protocol) client
 * Connects to a MediaMTX or compatible WHEP endpoint and returns the MediaStream.
 */
export async function connectWHEP(url) {
  const pc = new RTCPeerConnection({
    iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
  });

  pc.addTransceiver('video', { direction: 'recvonly' });
  pc.addTransceiver('audio', { direction: 'recvonly' });

  const offer = await pc.createOffer();
  await pc.setLocalDescription(offer);

  // Wait for ICE gathering, con tope: en red local/VPN los host candidates
  // bastan, y sin salida a internet el STUN puede demorar ~10s en expirar.
  await new Promise((resolve) => {
    if (pc.iceGatheringState === 'complete') {
      resolve();
      return;
    }
    const timeout = setTimeout(() => {
      pc.removeEventListener('icegatheringstatechange', check);
      resolve();
    }, 2000);
    const check = () => {
      if (pc.iceGatheringState === 'complete') {
        clearTimeout(timeout);
        pc.removeEventListener('icegatheringstatechange', check);
        resolve();
      }
    };
    pc.addEventListener('icegatheringstatechange', check);
  });

  const whepUrl = url.endsWith('/') ? `${url}whep` : `${url}/whep`;

  const res = await fetch(whepUrl, {
    method: 'POST',
    headers: { 'Content-Type': 'application/sdp' },
    body: pc.localDescription.sdp,
  });

  if (!res.ok) {
    pc.close();
    throw new Error(`WHEP error: ${res.status} ${res.statusText}`);
  }

  const answerSDP = await res.text();
  await pc.setRemoteDescription(new RTCSessionDescription({
    type: 'answer',
    sdp: answerSDP,
  }));

  // Collect the MediaStream from incoming tracks
  const stream = new MediaStream();
  pc.getReceivers().forEach((receiver) => {
    if (receiver.track) {
      stream.addTrack(receiver.track);
    }
  });

  return { pc, stream };
}
