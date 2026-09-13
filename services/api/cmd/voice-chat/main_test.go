package main

import "testing"

func TestServerPort(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int
		wantErr bool
	}{
		{name: "default", want: 8080},
		{name: "configured", value: "9090", want: 9090},
		{name: "not a number", value: "invalid", wantErr: true},
		{name: "too low", value: "0", wantErr: true},
		{name: "too high", value: "65536", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("PORT", test.value)
			got, err := serverPort()
			if (err != nil) != test.wantErr {
				t.Fatalf("serverPort() error = %v, wantErr %t", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("serverPort() = %d, want %d", got, test.want)
			}
		})
	}
}
