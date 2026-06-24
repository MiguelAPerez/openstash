package cli

import "testing"

func TestResolveMaxBodyBytes(t *testing.T) {
	t.Setenv("OPENSTASH_MAX_BODY_BYTES", "")

	// No flag, no env -> 0 (server applies its default).
	if got, err := resolveMaxBodyBytes(0); err != nil || got != 0 {
		t.Fatalf("flag=0 env=unset: got %d, err %v; want 0, nil", got, err)
	}

	// Flag takes precedence over env.
	t.Setenv("OPENSTASH_MAX_BODY_BYTES", "2048")
	if got, err := resolveMaxBodyBytes(4096); err != nil || got != 4096 {
		t.Fatalf("flag=4096 env=2048: got %d, err %v; want 4096, nil", got, err)
	}

	// Env used when flag is 0.
	if got, err := resolveMaxBodyBytes(0); err != nil || got != 2048 {
		t.Fatalf("flag=0 env=2048: got %d, err %v; want 2048, nil", got, err)
	}

	// Invalid env values are rejected.
	for _, bad := range []string{"abc", "0", "-1"} {
		t.Setenv("OPENSTASH_MAX_BODY_BYTES", bad)
		if _, err := resolveMaxBodyBytes(0); err == nil {
			t.Fatalf("env=%q: expected error, got nil", bad)
		}
	}
}
