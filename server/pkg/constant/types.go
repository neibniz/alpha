package constant

// Used for type cloning conversion
const (
	Int64   int64  = 0
	Uint32  uint32 = 0
	DevMode        = "dev"
)

// VerifyType is the type of verification code
type VerifyType uint8

const (
	Register VerifyType = iota + 1
	Security
)

func ParseVerifyType(i uint8) VerifyType {
	return VerifyType(i)
}

func (v VerifyType) String() string {
	switch v {
	case Register:
		return "register"
	case Security:
		return "security"
	default:
		return "unknown"
	}
}
