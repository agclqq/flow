package valueobject

// ParamType 表示流水线参数的类型。
type ParamType string

const (
	ParamTypeString  ParamType = "string"
	ParamTypeNumber  ParamType = "number"
	ParamTypeBoolean ParamType = "boolean"
)

// ParamType 的 tinyint 存储值。
// 0: string（字符串） 1: number（数值） 2: boolean（布尔）
const (
	ParamTypeStringInt  int8 = 0
	ParamTypeNumberInt  int8 = 1
	ParamTypeBooleanInt int8 = 2
)

// Int 返回参数类型的 tinyint 存储值。
func (t ParamType) Int() int8 {
	switch t {
	case ParamTypeNumber:
		return ParamTypeNumberInt
	case ParamTypeBoolean:
		return ParamTypeBooleanInt
	default:
		return ParamTypeStringInt
	}
}

// ParamTypeFromInt 将 tinyint 值转换为 ParamType。
func ParamTypeFromInt(v int8) ParamType {
	switch v {
	case ParamTypeNumberInt:
		return ParamTypeNumber
	case ParamTypeBooleanInt:
		return ParamTypeBoolean
	default:
		return ParamTypeString
	}
}

// Valid 校验参数类型是否合法。
func (t ParamType) Valid() bool {
	return t == ParamTypeString || t == ParamTypeNumber || t == ParamTypeBoolean
}

// String 返回参数类型的字符串表示。
func (t ParamType) String() string {
	return string(t)
}

// AuthType 表示裸机执行环境的认证方式。
type AuthType string

const (
	AuthTypePassword AuthType = "password" // 用户 + 密码
	AuthTypeSSHKey   AuthType = "sshkey"   // SSH 私钥
)

// AuthType 的 tinyint 存储值。
// 0: password（用户+密码） 1: sshkey（SSH 私钥）
const (
	AuthTypePasswordInt int8 = 0
	AuthTypeSSHKeyInt   int8 = 1
)

// Int 返回认证方式的 tinyint 存储值。
func (t AuthType) Int() int8 {
	if t == AuthTypeSSHKey {
		return AuthTypeSSHKeyInt
	}
	return AuthTypePasswordInt
}

// AuthTypeFromInt 将 tinyint 值转换为 AuthType。
func AuthTypeFromInt(v int8) AuthType {
	if v == AuthTypeSSHKeyInt {
		return AuthTypeSSHKey
	}
	return AuthTypePassword
}

// Valid 校验认证方式是否合法。
func (t AuthType) Valid() bool {
	return t == AuthTypePassword || t == AuthTypeSSHKey
}

// String 返回认证方式的字符串表示。
func (t AuthType) String() string {
	return string(t)
}
