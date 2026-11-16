package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUpgradeWelcomeText(t *testing.T) {
	cases := map[string]string{
		"Email: {{email}}\nPassword: {{password}}\n\nSegera ganti password setelah login.": "Email: {{email}}\nAtur password: {{link_set_password}}\n\nLink berlaku 72 jam dan hanya bisa dipakai 1 kali.",
		"Halo! Email: {{email}}, Password: {{password}}":                                   "Halo! Email: {{email}}, Atur password: {{link_set_password}}",
		"Kata sandi kamu {{password}}":                                                     "Kata sandi kamu {{link_set_password}}",
		"Already new: {{link_set_password}}":                                               "Already new: {{link_set_password}}",
	}
	for in, want := range cases {
		if got := upgradeWelcomeText(in); got != want {
			t.Errorf("upgradeWelcomeText(%q)\n got %q\nwant %q", in, got, want)
		}
	}
}

func TestUpgradeWelcomeVars(t *testing.T) {
	in := `[{"key": "nama_customer", "label": "Nama Customer"}, {"key": "email", "label": "Email"}, {"key": "password", "label": "Password"}]`
	got, err := upgradeWelcomeVars(in)
	if err != nil {
		t.Fatal(err)
	}
	var vars []map[string]string
	if err := json.Unmarshal([]byte(got), &vars); err != nil {
		t.Fatal(err)
	}
	if len(vars) != 3 || vars[2]["key"] != "link_set_password" || strings.Contains(got, `"password"`) {
		t.Errorf("password variable must become link_set_password, got %s", got)
	}

	again, _ := upgradeWelcomeVars(got)
	if again != got {
		t.Errorf("must be idempotent: %s then %s", got, again)
	}
}
