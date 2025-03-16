package v2

// OpType 操作类型
type OpType int

const (
	OpUnknown    OpType = iota // 未知, 初始状态
	OpConnect                  // 发送连接
	OpConnecting               // 连接中
	OpConnected                // 已连接
	OpRead                     // 读
	OpWrite                    // 写
	OpClose                    // 关闭
	OpClosing                  // 关闭中
	OpClosed                   // 已关闭
)
