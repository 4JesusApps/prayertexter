package domain_test

import (
	"testing"

	"github.com/4JesusApps/prayertexter/internal/domain"
)

func TestGenRandPhones(t *testing.T) {
	tests := []struct {
		name                  string
		phones                []string
		intercessorsPerPrayer int
		wantLen               int
		wantNil               bool
	}{
		{
			name:                  "empty list returns nil",
			phones:                []string{},
			intercessorsPerPrayer: 2,
			wantNil:               true,
		},
		{
			name:                  "fewer phones than requested returns all",
			phones:                []string{"+11111111111"},
			intercessorsPerPrayer: 3,
			wantLen:               1,
		},
		{
			name:                  "exact count returns all",
			phones:                []string{"+11111111111", "+12222222222"},
			intercessorsPerPrayer: 2,
			wantLen:               2,
		},
		{
			name:                  "more phones than requested returns requested count with unique values",
			phones:                []string{"+11111111111", "+12222222222", "+13333333333", "+14444444444"},
			intercessorsPerPrayer: 2,
			wantLen:               2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := &domain.IntercessorPhones{Phones: tt.phones}
			got := ip.GenRandPhones(tt.intercessorsPerPrayer)
			if tt.wantNil {
				if got != nil {
					t.Errorf("GenRandPhones() = %v, want nil", got)
				}
				return
			}
			if len(got) != tt.wantLen {
				t.Errorf("GenRandPhones() returned %d phones, want %d", len(got), tt.wantLen)
			}
			seen := make(map[string]bool)
			for _, p := range got {
				if seen[p] {
					t.Errorf("GenRandPhones() returned duplicate phone %s", p)
				}
				seen[p] = true
			}
		})
	}
}

func TestAddPhone(t *testing.T) {
	t.Run("blocked phones deduplicates", func(t *testing.T) {
		bp := &domain.BlockedPhones{}
		bp.AddPhone("+11111111111")
		bp.AddPhone("+11111111111")
		if len(bp.Phones) != 1 {
			t.Errorf("AddPhone() resulted in %d phones, want 1", len(bp.Phones))
		}
	})

	t.Run("intercessor phones deduplicates", func(t *testing.T) {
		ip := &domain.IntercessorPhones{}
		ip.AddPhone("+11111111111")
		ip.AddPhone("+11111111111")
		if len(ip.Phones) != 1 {
			t.Errorf("AddPhone() resulted in %d phones, want 1", len(ip.Phones))
		}
	})
}

func TestRemovePhone(t *testing.T) {
	t.Run("blocked phones removes target", func(t *testing.T) {
		bp := &domain.BlockedPhones{Phones: []string{"+11111111111", "+12222222222"}}
		bp.RemovePhone("+11111111111")
		if len(bp.Phones) != 1 || bp.Phones[0] != "+12222222222" {
			t.Errorf("RemovePhone() = %v, want [+12222222222]", bp.Phones)
		}
	})

	t.Run("intercessor phones removes target", func(t *testing.T) {
		ip := &domain.IntercessorPhones{Phones: []string{"+11111111111", "+12222222222"}}
		ip.RemovePhone("+11111111111")
		if len(ip.Phones) != 1 || ip.Phones[0] != "+12222222222" {
			t.Errorf("RemovePhone() = %v, want [+12222222222]", ip.Phones)
		}
	})
}
