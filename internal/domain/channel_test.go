package domain

import "testing"

func TestChannelIsRadio(t *testing.T) {
	cases := []struct {
		name    string
		ch      Channel
		want    bool
	}{
		{"BBC Radio 2 by name", Channel{Name: "BBC Radio 2"}, true},
		{"Classic FM by name", Channel{Name: "Classic FM"}, true},
		{"Capital FM by name", Channel{Name: "Capital FM"}, true},
		{"BBC 6 Music by name", Channel{Name: "BBC 6 Music"}, true},
		{"BBC R1X by name", Channel{Name: "BBC R1X"}, true},
		{"BBC R5L by name", Channel{Name: "BBC R5L"}, true},
		{"BBC Asian Net. by name", Channel{Name: "BBC Asian Net."}, true},
		{"BBC World Sv. by name", Channel{Name: "BBC World Sv."}, true},
		{"Radio by xmltv_id", Channel{Name: "Something", XmltvID: "SomeRadio.uk"}, true},
		{"BBC One is TV", Channel{Name: "BBC ONE"}, false},
		{"ITV is TV", Channel{Name: "ITV1"}, false},
		{"Sky Sports is TV", Channel{Name: "Sky Sports 1"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.ch.IsRadio(); got != tc.want {
				t.Errorf("IsRadio() = %v, want %v", got, tc.want)
			}
		})
	}
}
