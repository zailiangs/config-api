package sshconfig

import "testing"

func TestSetGlobalPort(t *testing.T) {
	tests := []struct {
		name  string
		input string
		port  int
		want  string
	}{
		{
			name:  "replace and remove duplicate",
			input: "# comment\nPort 22 # main\n  port 2222\nPermitRootLogin no\n",
			port:  18822,
			want:  "# comment\nPort 18822 # main\nPermitRootLogin no\n",
		},
		{
			name:  "insert before match",
			input: "# comment\nPermitRootLogin no\nMatch User deploy\n  Port 2200\n",
			port:  18822,
			want:  "# comment\nPermitRootLogin no\nPort 18822\nMatch User deploy\n  Port 2200\n",
		},
		{
			name:  "leave include and match alone",
			input: "Include /etc/ssh/sshd_config.d/*.conf\nPort 22\nMatch All\n Port 2200\n",
			port:  22,
			want:  "Include /etc/ssh/sshd_config.d/*.conf\nPort 22\nMatch All\n Port 2200\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := SetGlobalPort([]byte(tt.input), tt.port)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Fatalf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestSetGlobalPortRejectsInvalidPort(t *testing.T) {
	for _, port := range []int{0, 65536} {
		if _, _, err := SetGlobalPort(nil, port); err == nil {
			t.Fatalf("expected port %d to fail", port)
		}
	}
}
