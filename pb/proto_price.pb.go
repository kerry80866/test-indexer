// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.29.3
// source: proto_price.proto

package pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// 请求参数
type GetPriceHistoryRequest struct {
	state          protoimpl.MessageState `protogen:"open.v1"`
	ChainId        int32                  `protobuf:"varint,1,opt,name=chainId,proto3" json:"chainId,omitempty"`              // 链ID: 0=solana, 1=ethereum, ...
	TokenAddresses []string               `protobuf:"bytes,2,rep,name=tokenAddresses,proto3" json:"tokenAddresses,omitempty"` // token地址 (base58 string)
	FromTimestamp  int64                  `protobuf:"varint,3,opt,name=fromTimestamp,proto3" json:"fromTimestamp,omitempty"`  // 起始时间戳（秒）
	unknownFields  protoimpl.UnknownFields
	sizeCache      protoimpl.SizeCache
}

func (x *GetPriceHistoryRequest) Reset() {
	*x = GetPriceHistoryRequest{}
	mi := &file_proto_price_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetPriceHistoryRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetPriceHistoryRequest) ProtoMessage() {}

func (x *GetPriceHistoryRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_price_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetPriceHistoryRequest.ProtoReflect.Descriptor instead.
func (*GetPriceHistoryRequest) Descriptor() ([]byte, []int) {
	return file_proto_price_proto_rawDescGZIP(), []int{0}
}

func (x *GetPriceHistoryRequest) GetChainId() int32 {
	if x != nil {
		return x.ChainId
	}
	return 0
}

func (x *GetPriceHistoryRequest) GetTokenAddresses() []string {
	if x != nil {
		return x.TokenAddresses
	}
	return nil
}

func (x *GetPriceHistoryRequest) GetFromTimestamp() int64 {
	if x != nil {
		return x.FromTimestamp
	}
	return 0
}

// token 每个时间点的价格（单位是 USD）
type TokenPricePoint struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Timestamp     int64                  `protobuf:"varint,1,opt,name=timestamp,proto3" json:"timestamp,omitempty"` // 打点时间戳（秒）
	PriceUsd      float64                `protobuf:"fixed64,2,opt,name=priceUsd,proto3" json:"priceUsd,omitempty"`  // 单价（1 token 对应的 USD 价格）
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *TokenPricePoint) Reset() {
	*x = TokenPricePoint{}
	mi := &file_proto_price_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *TokenPricePoint) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TokenPricePoint) ProtoMessage() {}

func (x *TokenPricePoint) ProtoReflect() protoreflect.Message {
	mi := &file_proto_price_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TokenPricePoint.ProtoReflect.Descriptor instead.
func (*TokenPricePoint) Descriptor() ([]byte, []int) {
	return file_proto_price_proto_rawDescGZIP(), []int{1}
}

func (x *TokenPricePoint) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

func (x *TokenPricePoint) GetPriceUsd() float64 {
	if x != nil {
		return x.PriceUsd
	}
	return 0
}

// token 的时间序列
type TokenPriceHistory struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Points        []*TokenPricePoint     `protobuf:"bytes,1,rep,name=points,proto3" json:"points,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *TokenPriceHistory) Reset() {
	*x = TokenPriceHistory{}
	mi := &file_proto_price_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *TokenPriceHistory) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TokenPriceHistory) ProtoMessage() {}

func (x *TokenPriceHistory) ProtoReflect() protoreflect.Message {
	mi := &file_proto_price_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TokenPriceHistory.ProtoReflect.Descriptor instead.
func (*TokenPriceHistory) Descriptor() ([]byte, []int) {
	return file_proto_price_proto_rawDescGZIP(), []int{2}
}

func (x *TokenPriceHistory) GetPoints() []*TokenPricePoint {
	if x != nil {
		return x.Points
	}
	return nil
}

// 返回值：tokenAddress(base58 string) -> 时间序列
type GetPriceHistoryResponse struct {
	state         protoimpl.MessageState        `protogen:"open.v1"`
	Prices        map[string]*TokenPriceHistory `protobuf:"bytes,1,rep,name=prices,proto3" json:"prices,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetPriceHistoryResponse) Reset() {
	*x = GetPriceHistoryResponse{}
	mi := &file_proto_price_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetPriceHistoryResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetPriceHistoryResponse) ProtoMessage() {}

func (x *GetPriceHistoryResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_price_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetPriceHistoryResponse.ProtoReflect.Descriptor instead.
func (*GetPriceHistoryResponse) Descriptor() ([]byte, []int) {
	return file_proto_price_proto_rawDescGZIP(), []int{3}
}

func (x *GetPriceHistoryResponse) GetPrices() map[string]*TokenPriceHistory {
	if x != nil {
		return x.Prices
	}
	return nil
}

var File_proto_price_proto protoreflect.FileDescriptor

const file_proto_price_proto_rawDesc = "" +
	"\n" +
	"\x11proto_price.proto\x12\x05quote\"\x80\x01\n" +
	"\x16GetPriceHistoryRequest\x12\x18\n" +
	"\achainId\x18\x01 \x01(\x05R\achainId\x12&\n" +
	"\x0etokenAddresses\x18\x02 \x03(\tR\x0etokenAddresses\x12$\n" +
	"\rfromTimestamp\x18\x03 \x01(\x03R\rfromTimestamp\"K\n" +
	"\x0fTokenPricePoint\x12\x1c\n" +
	"\ttimestamp\x18\x01 \x01(\x03R\ttimestamp\x12\x1a\n" +
	"\bpriceUsd\x18\x02 \x01(\x01R\bpriceUsd\"C\n" +
	"\x11TokenPriceHistory\x12.\n" +
	"\x06points\x18\x01 \x03(\v2\x16.quote.TokenPricePointR\x06points\"\xb2\x01\n" +
	"\x17GetPriceHistoryResponse\x12B\n" +
	"\x06prices\x18\x01 \x03(\v2*.quote.GetPriceHistoryResponse.PricesEntryR\x06prices\x1aS\n" +
	"\vPricesEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12.\n" +
	"\x05value\x18\x02 \x01(\v2\x18.quote.TokenPriceHistoryR\x05value:\x028\x012`\n" +
	"\fPriceService\x12P\n" +
	"\x0fGetPriceHistory\x12\x1d.quote.GetPriceHistoryRequest\x1a\x1e.quote.GetPriceHistoryResponseB\x15Z\x13dex-quote-svc/pb;pbb\x06proto3"

var (
	file_proto_price_proto_rawDescOnce sync.Once
	file_proto_price_proto_rawDescData []byte
)

func file_proto_price_proto_rawDescGZIP() []byte {
	file_proto_price_proto_rawDescOnce.Do(func() {
		file_proto_price_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proto_price_proto_rawDesc), len(file_proto_price_proto_rawDesc)))
	})
	return file_proto_price_proto_rawDescData
}

var file_proto_price_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_proto_price_proto_goTypes = []any{
	(*GetPriceHistoryRequest)(nil),  // 0: quote.GetPriceHistoryRequest
	(*TokenPricePoint)(nil),         // 1: quote.TokenPricePoint
	(*TokenPriceHistory)(nil),       // 2: quote.TokenPriceHistory
	(*GetPriceHistoryResponse)(nil), // 3: quote.GetPriceHistoryResponse
	nil,                             // 4: quote.GetPriceHistoryResponse.PricesEntry
}
var file_proto_price_proto_depIdxs = []int32{
	1, // 0: quote.TokenPriceHistory.points:type_name -> quote.TokenPricePoint
	4, // 1: quote.GetPriceHistoryResponse.prices:type_name -> quote.GetPriceHistoryResponse.PricesEntry
	2, // 2: quote.GetPriceHistoryResponse.PricesEntry.value:type_name -> quote.TokenPriceHistory
	0, // 3: quote.PriceService.GetPriceHistory:input_type -> quote.GetPriceHistoryRequest
	3, // 4: quote.PriceService.GetPriceHistory:output_type -> quote.GetPriceHistoryResponse
	4, // [4:5] is the sub-list for method output_type
	3, // [3:4] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_proto_price_proto_init() }
func file_proto_price_proto_init() {
	if File_proto_price_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proto_price_proto_rawDesc), len(file_proto_price_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_price_proto_goTypes,
		DependencyIndexes: file_proto_price_proto_depIdxs,
		MessageInfos:      file_proto_price_proto_msgTypes,
	}.Build()
	File_proto_price_proto = out.File
	file_proto_price_proto_goTypes = nil
	file_proto_price_proto_depIdxs = nil
}
