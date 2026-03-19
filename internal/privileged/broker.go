package privileged

import (
	"knit/internal/audio"
	"knit/internal/capture"
	"knit/internal/companion"
	"knit/internal/session"
)

// CaptureBroker is the trust boundary between privileged local capture modules
// and the less-trusted UI/plugin-facing HTTP layer.
type CaptureBroker struct {
	capture *capture.Manager
	pointer *companion.Tracker
	audio   *audio.Controller
}

func NewCaptureBroker(captureManager *capture.Manager, pointerTracker *companion.Tracker, audioController *audio.Controller) *CaptureBroker {
	return &CaptureBroker{
		capture: captureManager,
		pointer: pointerTracker,
		audio:   audioController,
	}
}

func (b *CaptureBroker) Start() {
	if b == nil || b.capture == nil {
		return
	}
	b.capture.Start()
}

func (b *CaptureBroker) Pause() {
	if b == nil || b.capture == nil {
		return
	}
	b.capture.Pause()
}

func (b *CaptureBroker) Stop() {
	if b == nil || b.capture == nil {
		return
	}
	b.capture.Stop()
}

func (b *CaptureBroker) State() capture.State {
	if b == nil || b.capture == nil {
		return capture.StateInactive
	}
	return b.capture.State()
}

func (b *CaptureBroker) SetSourceStatus(source, status, reason string) {
	if b == nil || b.capture == nil {
		return
	}
	b.capture.SetSourceStatus(source, status, reason)
}

func (b *CaptureBroker) SourceStatuses() map[string]capture.SourceState {
	if b == nil || b.capture == nil {
		return map[string]capture.SourceState{}
	}
	return b.capture.SourceStatuses()
}

func (b *CaptureBroker) ReducedCapabilities() []string {
	if b == nil || b.capture == nil {
		return nil
	}
	return b.capture.ReducedCapabilities()
}

func (b *CaptureBroker) AudioState() audio.State {
	if b == nil || b.audio == nil {
		return audio.State{}
	}
	return b.audio.State()
}

func (b *CaptureBroker) AudioDevices() []audio.Device {
	if b == nil || b.audio == nil {
		return nil
	}
	return b.audio.Devices()
}

func (b *CaptureBroker) SetAudioDevices(devices []audio.Device) {
	if b == nil || b.audio == nil {
		return
	}
	b.audio.SetDevices(devices)
}

func (b *CaptureBroker) ConfigureAudio(cfg audio.Config) audio.State {
	if b == nil || b.audio == nil {
		return audio.State{}
	}
	return b.audio.Configure(cfg)
}

func (b *CaptureBroker) UpdateAudioLevel(level float64) audio.State {
	if b == nil || b.audio == nil {
		return audio.State{}
	}
	return b.audio.UpdateLevel(level)
}

func (b *CaptureBroker) AddPointer(evt companion.PointerEvent) {
	if b == nil || b.pointer == nil {
		return
	}
	b.pointer.Add(evt)
}

func (b *CaptureBroker) PointerSnapshot(sessionID string) (session.PointerCtx, []session.PointerSample) {
	if b == nil || b.pointer == nil {
		return session.PointerCtx{}, nil
	}
	return b.pointer.Snapshot(sessionID)
}

func (b *CaptureBroker) PointerReplaySnapshot(sessionID string) []session.ReplayStep {
	if b == nil || b.pointer == nil {
		return nil
	}
	return b.pointer.ReplaySnapshot(sessionID)
}
