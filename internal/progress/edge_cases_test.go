package progress_test

import (
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/progress"
	"testing"
)

func TestEffectiveGainCapsAndNeverLowersSkill(t *testing.T) {
	for _, tc := range []struct{ current, gain, cap, want int }{
		{1, 2, 4, 2}, {3, 3, 4, 1}, {4, 10, 10, 1}, {4, 2, 4, 0}, {5, 2, 3, 0},
	} {
		got := progress.EffectiveGain(tc.current, domain.EventSkillGain{Gain: tc.gain, MaxLevel: tc.cap})
		if got != tc.want {
			t.Errorf("%+v: got %d", tc, got)
		}
	}
}
