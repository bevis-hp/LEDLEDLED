package pixel

// PowerModel estimates a strip's current draw from its encoded DMX bytes.
//
// LED current is proportional to PWM duty, and the encoded byte (after
// gamma) is the duty, so the estimate is simply the sum of all channel
// bytes scaled by the full-on current of one channel, plus a quiescent
// draw per pixel for the driver chip.
//
// Typical values (per pixel = per addressable unit):
//
//	WS2812B 5V        ~20 mA per channel (60 mA white), ~1 mA idle
//	SK6812 RGBW 5V    ~20 mA per channel (80 mA all on)
//	WS2811 12V        ~20 mA per channel per 3-LED segment, ~2 mA idle
//	WS2815 12V        ~12 mA per channel
type PowerModel struct {
	Volts               float64
	MilliampsPerChannel float64
	IdleMilliamps       float64 // per pixel
}

// Milliamps estimates the draw of pixels whose channel bytes are data.
func (m PowerModel) Milliamps(data []byte, pixels int) float64 {
	sum := 0
	for _, b := range data {
		sum += int(b)
	}
	return float64(sum)/255*m.MilliampsPerChannel + float64(pixels)*m.IdleMilliamps
}

// Limit scales data in place so the estimated draw doesn't exceed maxMA.
// It returns the draw before and after limiting and the scale applied
// (1 = unchanged). maxMA <= 0 disables limiting.
func (m PowerModel) Limit(data []byte, pixels int, maxMA float64) (before, after, scale float64) {
	before = m.Milliamps(data, pixels)
	if maxMA <= 0 || before <= maxMA {
		return before, before, 1
	}
	idle := float64(pixels) * m.IdleMilliamps
	scale = 0
	if before > idle {
		scale = max(0, (maxMA-idle)/(before-idle))
	}
	for i, b := range data {
		data[i] = uint8(float64(b) * scale) // truncate so we stay under
	}
	return before, m.Milliamps(data, pixels), scale
}
