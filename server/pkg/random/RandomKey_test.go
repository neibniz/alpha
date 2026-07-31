package random

import (
	"math/rand"
	"testing"
	"time"

	"github.com/neibniz/alpha/server/pkg/snowflake"

	"github.com/stretchr/testify/assert"
)

func TestEncodeBase62(t *testing.T) {
	start := 1112275807
	length := 1558080
	n := length + start
	// n := 328564998144
	m := make(map[string]struct{})
	// m := make(map[string]struct{}, length)
	var inviteCode string
	for i := start; i < n; i++ {
		// inviteCode = EncodeBase36(int64(i))
		inviteCode = EncodeBase36(snowflake.GetID())
		if v, ok := m[inviteCode]; ok {
			t.Fatal(v, inviteCode)
		}
		m[inviteCode] = struct{}{}
	}
	t.Log(inviteCode)

	assert.Equal(t, length, len(m))
}

func TestInt64ToDashedString(t *testing.T) {
	type args struct {
		strNum string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{
			name: "",
			args: args{
				strNum: "123",
			},
			want: "123",
		},
		{
			name: "",
			args: args{
				strNum: "1234",
			},
			want: "1234",
		},
		{
			name: "",
			args: args{
				strNum: "12345",
			},
			want: "1234-5",
		},
		{
			name: "",
			args: args{
				strNum: "12345678",
			},
			want: "1234-5678",
		},
		{
			name: "",
			args: args{
				strNum: "123456789",
			},
			want: "1234-5678-9",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, StrToDashedString(tt.args.strNum), "StrToDashedString(%v)", tt.args.strNum)
		})
	}
}

func TestKeyPreallocatesAndKeepsAlphabetShape(t *testing.T) {
	got := Key(16, 0)
	assert.Len(t, got, 16)
	for _, ch := range got {
		assert.True(t, ch >= '0' && ch <= '9', "Key(16, 0) contains %q", ch)
	}

	got = KeyNew(16, 2)
	assert.Len(t, got, 16)
	for _, ch := range got {
		assert.True(t, (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9'), "KeyNew(16, 2) contains %q", ch)
	}
}

func TestKeyKeepsEmptyShapeForNonPositiveLength(t *testing.T) {
	assert.Empty(t, Key(0, 1))
	assert.Empty(t, KeyNew(-1, 1))
}

// ShuffleString shuffles the characters in a string.
func ShuffleString(s string) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	runes := []rune(s)
	for i := range runes {
		j := r.Intn(i + 1)
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
