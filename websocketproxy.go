package main

import (
	"errors"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

type ProtoJSONConverter interface {
	JSONToProto(jsonData []byte) (proto.Message, error)
	ProtoToJSON(protoData []byte) ([]byte, error)
}

type ProtobufProxy struct {
	wsConn       *websocket.Conn
	tcpConn      net.Conn
	msgConverter ProtoJSONConverter
}

func (p *ProtobufProxy) handleProtobufProxy() {
	for {
		// 从 WebSocket 客户端接收数据
		_, data, err := p.wsConn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		// 将接收到的数据转换为 protobuf
		msg, err := p.msgConverter.JSONToProto(data)
		if err != nil {
			log.Println(err)
			continue
		}

		// 发送 protobuf 数据到 TCP 后端服务
		data, err = proto.Marshal(msg)
		if err != nil {
			log.Println(err)
			continue
		}
		if _, err := p.tcpConn.Write(data); err != nil {
			log.Println(err)
			continue
		}

		// 从 TCP 后端服务接收数据
		data = make([]byte, 1024)
		n, err := p.tcpConn.Read(data)
		if err != nil {
			log.Println(err)
			continue
		}

		// 将接收到的数据转换为 JSON
		jsonData, err := p.msgConverter.ProtoToJSON(data[:n])
		if err != nil {
			log.Println(err)
			continue
		}

		// 发送 JSON 数据到 WebSocket 客户端
		if err := p.wsConn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
			log.Println(err)
			continue
		}
	}
}

type DefaultProtoJSONConverter struct{}

func (c *DefaultProtoJSONConverter) JSONToProto(jsonData []byte) (proto.Message, error) {
	// TODO: 实现 JSON 数据转换为 protobuf 数据的逻辑
	return nil, errors.New("not implemented")
}

func (c *DefaultProtoJSONConverter) ProtoToJSON(protoData []byte) ([]byte, error) {
	// TODO: 实现 protobuf 数据转换为 JSON 数据的逻辑
	return nil, errors.New("not implemented")
}

func main() {
	// 创建 WebSocket 服务器
	wsServer := &websocket.Server{
		Handshake: func(config *websocket.Config, req *http.Request) error {
			// TODO: 验证请求是否合法
			return nil
		},
	}

	// 监听 TCP 连接
	tcpListener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer tcpListener.Close()

	// 循环接受 TCP 连接，并启动代理服务
	for {
		tcpConn, err := tcpListener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		// 从 TCP 连接中读取前 4 个字节，解析出 WebSocket 的握手协议
		handshake := make([]byte, 4)
		if _, err := tcpConn.Read(handshake); err != nil {
			log.Println(err)
			continue
		}

		// 将 TCP 连接升级为 WebSocket 连接
		wsConn, _ := wsServer.Upgrade(tcpConn, nil, nil)

		// 创建代理服务，并启动代理服务
		proxy := &ProtobufProxy{
			wsConn:       wsConn,
			tcpConn:      tcpConn,
			msgConverter: &DefaultProtoJSONConverter{},
		}
		go proxy.handleProtobufProxy()
	}
}
