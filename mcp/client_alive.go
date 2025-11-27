package mcp

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"
)

type client3EAlive struct {
	conn net.Conn
	// PLC address
	tcpAddr *net.TCPAddr
	// PLC station
	stn *station
	// 用于保护并发访问
	mu sync.Mutex
}

// 获取连接，如果连接不存在或已关闭则重新创建
func (c *client3EAlive) getConnection() (net.Conn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		// 检查连接是否有效
		_, err := c.conn.Write([]byte{}) // 发送空字节测试连接
		if err == nil {
			return c.conn, nil
		}
		// 连接无效，关闭它
		c.conn.Close()
		c.conn = nil
	}

	// 创建新连接
	conn, err := net.DialTCP("tcp", nil, c.tcpAddr)
	if err != nil {
		return nil, err
	}
	c.conn = conn
	return conn, nil
}

// Close关闭连接
func (c *client3EAlive) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

// HealthCheck实现Client接口的HealthCheck方法
func (c *client3EAlive) HealthCheck() error {
	requestStr := c.stn.BuildHealthCheckRequest()

	// 二进制协议
	payload, err := hex.DecodeString(requestStr)
	if err != nil {
		return err
	}

	conn, err := c.getConnection()
	if err != nil {
		return err
	}

	// 发送消息
	if _, err = conn.Write(payload); err != nil {
		// 连接可能已断开，下一次操作会重新创建
		c.mu.Lock()
		c.conn.Close()
		c.conn = nil
		c.mu.Unlock()
		return err
	}

	// 接收消息
	readBuff := make([]byte, 30)
	readLen, err := conn.Read(readBuff)
	if err != nil {
		// 连接可能已断开，下一次操作会重新创建
		c.mu.Lock()
		c.conn.Close()
		c.conn = nil
		c.mu.Unlock()
		return err
	}

	resp := readBuff[:readLen]

	if readLen != 18 {
		return errors.New("plc connect test is fail: return length is [" + fmt.Sprintf("%X", resp) + "]")
	}

	// decodeString is 折返しデータ数ヘッダ[1byte]
	if "0500" != fmt.Sprintf("%X", resp[11:13]) {
		return errors.New("plc connect test is fail: return header is [" + fmt.Sprintf("%X", resp[11:13]) + "]")
	}

	//  折返しデータ[5byte]=ABCDE
	if "4142434445" != fmt.Sprintf("%X", resp[13:18]) {
		return errors.New("plc connect test is fail: return body is [" + fmt.Sprintf("%X", resp[13:18]) + "]")
	}

	return nil
}

// Read实现Client接口的Read方法
func (c *client3EAlive) Read(deviceName string, offset, numPoints int64) ([]byte, error) {
	requestStr := c.stn.BuildReadRequest(deviceName, offset, numPoints)

	// 二进制协议
	payload, err := hex.DecodeString(requestStr)
	if err != nil {
		return nil, err
	}

	conn, err := c.getConnection()
	if err != nil {
		return nil, err
	}

	// 发送消息
	if _, err = conn.Write(payload); err != nil {
		// 连接可能已断开，下一次操作会重新创建
		c.mu.Lock()
		c.conn.Close()
		c.conn = nil
		c.mu.Unlock()
		return nil, err
	}

	// 接收消息
	readBuff := make([]byte, 22+2*numPoints) // 22是响应头大小
	readLen, err := conn.Read(readBuff)
	if err != nil {
		// 连接可能已断开，下一次操作会重新创建
		c.mu.Lock()
		c.conn.Close()
		c.conn = nil
		c.mu.Unlock()
		return nil, err
	}

	return readBuff[:readLen], nil
}

// BitRead实现Client接口的BitRead方法
func (c *client3EAlive) BitRead(deviceName string, offset, numPoints int64) ([]byte, error) {
	requestStr := c.stn.BuildBitReadRequest(deviceName, offset, numPoints)

	// 二进制协议
	payload, err := hex.DecodeString(requestStr)
	if err != nil {
		return nil, err
	}

	conn, err := c.getConnection()
	if err != nil {
		return nil, err
	}

	// 发送消息
	if _, err = conn.Write(payload); err != nil {
		// 连接可能已断开，下一次操作会重新创建
		c.mu.Lock()
		c.conn.Close()
		c.conn = nil
		c.mu.Unlock()
		return nil, err
	}

	// 接收消息
	readBuff := make([]byte, 22+2*numPoints) // 22是响应头大小
	readLen, err := conn.Read(readBuff)
	if err != nil {
		// 连接可能已断开，下一次操作会重新创建
		c.mu.Lock()
		c.conn.Close()
		c.conn = nil
		c.mu.Unlock()
		return nil, err
	}

	return readBuff[:readLen], nil
}

// Write实现Client接口的Write方法
func (c *client3EAlive) Write(deviceName string, offset, numPoints int64, writeData []byte) ([]byte, error) {
	requestStr := c.stn.BuildWriteRequest(deviceName, offset, numPoints, writeData)

	// 二进制协议
	payload, err := hex.DecodeString(requestStr)
	if err != nil {
		return nil, err
	}

	conn, err := c.getConnection()
	if err != nil {
		return nil, err
	}

	// 发送消息
	if _, err = conn.Write(payload); err != nil {
		// 连接可能已断开，下一次操作会重新创建
		c.mu.Lock()
		c.conn.Close()
		c.conn = nil
		c.mu.Unlock()
		return nil, err
	}

	// 接收消息
	readBuff := make([]byte, 22) // 22是响应头大小
	readLen, err := conn.Read(readBuff)
	if err != nil {
		// 连接可能已断开，下一次操作会重新创建
		c.mu.Lock()
		c.conn.Close()
		c.conn = nil
		c.mu.Unlock()
		return nil, err
	}

	return readBuff[:readLen], nil
}

// New3EAliveClient创建一个新的保持长连接的3E帧MCP客户端
func New3EAliveClient(host string, port int, stn *station) (Client, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%v:%v", host, port))
	if err != nil {
		return nil, err
	}
	return &client3EAlive{tcpAddr: tcpAddr, stn: stn}, nil
}
