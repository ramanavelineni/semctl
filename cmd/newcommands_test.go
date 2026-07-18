package cmd

import "testing"

// The schedule/token/event/info commands are thin closures over the tested
// resource helpers; assert they are registered with the expected shape.
func TestNewResourceCommandsRegistered(t *testing.T) {
	want := map[string][]string{
		"schedule": {"list", "show", "create", "update", "delete"},
		"token":    {"list", "create", "delete"},
		"event":    {"list"},
		"info":     nil,
	}
	for name, subs := range want {
		found := false
		for _, c := range rootCmd.Commands() {
			if c.Name() != name {
				continue
			}
			found = true
			for _, sub := range subs {
				ok := false
				for _, sc := range c.Commands() {
					if sc.Name() == sub {
						ok = true
						break
					}
				}
				if !ok {
					t.Errorf("%s %s not registered", name, sub)
				}
			}
		}
		if !found {
			t.Errorf("command %q not registered", name)
		}
	}
}
