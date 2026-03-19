package audio

import "testing"

func TestControllerDefaults(t *testing.T) {
	c := NewController()
	st := c.State()
	if st.Mode != ModeAlwaysOn {
		t.Fatalf("expected default mode always_on, got %q", st.Mode)
	}
	if st.InputDeviceID == "" {
		t.Fatalf("expected default input device")
	}
	if len(c.Devices()) == 0 {
		t.Fatalf("expected default device list")
	}
}

func TestConfigureAndLevelValidation(t *testing.T) {
	c := NewController()
	muted := true
	paused := true
	st := c.Configure(Config{
		Mode:          ModeAlwaysOn,
		InputDeviceID: "mic-2",
		Muted:         &muted,
		Paused:        &paused,
		LevelMin:      0.05,
		LevelMax:      0.80,
	})
	if st.Mode != ModeAlwaysOn {
		t.Fatalf("expected always_on mode")
	}
	if !st.Muted || !st.Paused {
		t.Fatalf("expected muted + paused set true")
	}
	if st.InputDeviceID != "mic-2" {
		t.Fatalf("expected device mic-2, got %q", st.InputDeviceID)
	}

	st = c.UpdateLevel(0.02)
	if st.LevelValid {
		t.Fatalf("expected level invalid below threshold")
	}
	st = c.UpdateLevel(0.25)
	if !st.LevelValid {
		t.Fatalf("expected level valid in threshold range")
	}
}

func TestSetDevicesNormalizesAndKeepsSelection(t *testing.T) {
	c := NewController()
	c.Configure(Config{InputDeviceID: "d2"})
	c.SetDevices([]Device{
		{ID: "d1", Label: "Primary"},
		{ID: "d2", Label: "Secondary"},
	})
	st := c.State()
	if st.InputDeviceID != "d2" {
		t.Fatalf("expected selected device to remain d2, got %q", st.InputDeviceID)
	}
	if st.InputDeviceName != "Secondary" {
		t.Fatalf("expected selected device label, got %q", st.InputDeviceName)
	}
}
