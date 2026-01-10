package exchange

import (
	"testing"
	"time"
)

func TestTimeframe(t *testing.T) {
	tests := []struct {
		tf      Timeframe
		minutes int
		valid   bool
	}{
		{Timeframe1m, 1, true},
		{Timeframe5m, 5, true},
		{Timeframe15m, 15, true},
		{Timeframe1h, 60, true},
		{Timeframe4h, 240, true},
		{Timeframe1d, 1440, true},
		{Timeframe("invalid"), 0, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.tf), func(t *testing.T) {
			// Test ToMinutes
			if got := tt.tf.ToMinutes(); got != tt.minutes {
				t.Errorf("ToMinutes() = %d, want %d", got, tt.minutes)
			}

			// Test ToDuration
			expectedDuration := time.Duration(tt.minutes) * time.Minute
			if got := tt.tf.ToDuration(); got != expectedDuration {
				t.Errorf("ToDuration() = %v, want %v", got, expectedDuration)
			}

			// Test IsValid
			if got := tt.tf.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestCandle(t *testing.T) {
	now := time.Now()
	candle := Candle{
		Timestamp: now,
		Open:      100.0,
		High:      110.0,
		Low:       95.0,
		Close:     105.0,
		Volume:    1000.0,
	}

	if candle.Timestamp != now {
		t.Errorf("Timestamp mismatch")
	}
	if candle.Open != 100.0 {
		t.Errorf("Open = %f, want 100.0", candle.Open)
	}
	if candle.High != 110.0 {
		t.Errorf("High = %f, want 110.0", candle.High)
	}
	if candle.Low != 95.0 {
		t.Errorf("Low = %f, want 95.0", candle.Low)
	}
	if candle.Close != 105.0 {
		t.Errorf("Close = %f, want 105.0", candle.Close)
	}
	if candle.Volume != 1000.0 {
		t.Errorf("Volume = %f, want 1000.0", candle.Volume)
	}
}
