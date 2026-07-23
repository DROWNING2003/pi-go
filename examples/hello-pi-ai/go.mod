module github.com/DROWNING2003/pi-go/examples/hello-pi-ai

go 1.26.1

require github.com/DROWNING2003/pi-go/packages/ai v0.1.0

// 本地开发：取消注释下面这行
replace github.com/DROWNING2003/pi-go/packages/ai => ../../packages/ai

// 外部用户：注释掉 replace，直接 go get
// go get github.com/DROWNING2003/pi-go/packages/ai@v0.1.0
