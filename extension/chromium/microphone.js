const enable = document.getElementById("enable");
const result = document.getElementById("result");
enable.onclick = async () => {
  enable.disabled = true;
  let stream;
  try {
    stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: false });
    result.textContent = "Microphone access allowed. Return to Knit and press the microphone button to record. You can close this tab.";
  } catch (_) {
    result.textContent = "Microphone access was not allowed. Check the Chrome and system microphone settings below, then try again.";
  } finally {
    if (stream) stream.getTracks().forEach(track => track.stop());
    enable.disabled = false;
  }
};
