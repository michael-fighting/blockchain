// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v4.23.1
// source: message.proto

package pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// payload type
type RBC_Type int32

const (
	RBC_VAL   RBC_Type = 0
	RBC_ECHO  RBC_Type = 1
	RBC_READY RBC_Type = 2
)

// Enum value maps for RBC_Type.
var (
	RBC_Type_name = map[int32]string{
		0: "VAL",
		1: "ECHO",
		2: "READY",
	}
	RBC_Type_value = map[string]int32{
		"VAL":   0,
		"ECHO":  1,
		"READY": 2,
	}
)

func (x RBC_Type) Enum() *RBC_Type {
	p := new(RBC_Type)
	*p = x
	return p
}

func (x RBC_Type) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (RBC_Type) Descriptor() protoreflect.EnumDescriptor {
	return file_message_proto_enumTypes[0].Descriptor()
}

func (RBC_Type) Type() protoreflect.EnumType {
	return &file_message_proto_enumTypes[0]
}

func (x RBC_Type) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use RBC_Type.Descriptor instead.
func (RBC_Type) EnumDescriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{1, 0}
}

// payload type
type BBA_Type int32

const (
	BBA_BVAL       BBA_Type = 0
	BBA_AUX        BBA_Type = 1
	BBA_MAIN_VOTE  BBA_Type = 2
	BBA_FINAL_VOTE BBA_Type = 3
	// FIXME 暂时将 COIN_SIG_SHARE 类型放到BA中
	BBA_COIN_SIG_SHARE BBA_Type = 4
)

// Enum value maps for BBA_Type.
var (
	BBA_Type_name = map[int32]string{
		0: "BVAL",
		1: "AUX",
		2: "MAIN_VOTE",
		3: "FINAL_VOTE",
		4: "COIN_SIG_SHARE",
	}
	BBA_Type_value = map[string]int32{
		"BVAL":           0,
		"AUX":            1,
		"MAIN_VOTE":      2,
		"FINAL_VOTE":     3,
		"COIN_SIG_SHARE": 4,
	}
)

func (x BBA_Type) Enum() *BBA_Type {
	p := new(BBA_Type)
	*p = x
	return p
}

func (x BBA_Type) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (BBA_Type) Descriptor() protoreflect.EnumDescriptor {
	return file_message_proto_enumTypes[1].Descriptor()
}

func (BBA_Type) Type() protoreflect.EnumType {
	return &file_message_proto_enumTypes[1]
}

func (x BBA_Type) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use BBA_Type.Descriptor instead.
func (BBA_Type) EnumDescriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{2, 0}
}

// CE消息的类型表示，SHARE表示为份额消息
type CE_Type int32

const (
	CE_SHARE CE_Type = 0
)

// Enum value maps for CE_Type.
var (
	CE_Type_name = map[int32]string{
		0: "SHARE",
	}
	CE_Type_value = map[string]int32{
		"SHARE": 0,
	}
)

func (x CE_Type) Enum() *CE_Type {
	p := new(CE_Type)
	*p = x
	return p
}

func (x CE_Type) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (CE_Type) Descriptor() protoreflect.EnumDescriptor {
	return file_message_proto_enumTypes[2].Descriptor()
}

func (CE_Type) Type() protoreflect.EnumType {
	return &file_message_proto_enumTypes[2]
}

func (x CE_Type) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use CE_Type.Descriptor instead.
func (CE_Type) EnumDescriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{3, 0}
}

type Message struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The signature of public key
	Signature []byte `protobuf:"bytes,1,opt,name=signature,proto3" json:"signature,omitempty"`
	// The address of source member who sent this message first
	Proposer string `protobuf:"bytes,2,opt,name=proposer,proto3" json:"proposer,omitempty"`
	// The address of sender member who sent this message
	Sender string `protobuf:"bytes,3,opt,name=sender,proto3" json:"sender,omitempty"`
	// The time when the source sends
	Timestamp *timestamppb.Timestamp `protobuf:"bytes,4,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	// message's epoch
	Epoch uint64 `protobuf:"varint,5,opt,name=epoch,proto3" json:"epoch,omitempty"`
	// Types that are assignable to Payload:
	//
	//	*Message_Rbc
	//	*Message_Bba
	//	*Message_Ce
	//	*Message_Ds
	Payload isMessage_Payload `protobuf_oneof:"payload"`
}

func (x *Message) Reset() {
	*x = Message{}
	if protoimpl.UnsafeEnabled {
		mi := &file_message_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Message) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Message) ProtoMessage() {}

func (x *Message) ProtoReflect() protoreflect.Message {
	mi := &file_message_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Message.ProtoReflect.Descriptor instead.
func (*Message) Descriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{0}
}

func (x *Message) GetSignature() []byte {
	if x != nil {
		return x.Signature
	}
	return nil
}

func (x *Message) GetProposer() string {
	if x != nil {
		return x.Proposer
	}
	return ""
}

func (x *Message) GetSender() string {
	if x != nil {
		return x.Sender
	}
	return ""
}

func (x *Message) GetTimestamp() *timestamppb.Timestamp {
	if x != nil {
		return x.Timestamp
	}
	return nil
}

func (x *Message) GetEpoch() uint64 {
	if x != nil {
		return x.Epoch
	}
	return 0
}

func (m *Message) GetPayload() isMessage_Payload {
	if m != nil {
		return m.Payload
	}
	return nil
}

func (x *Message) GetRbc() *RBC {
	if x, ok := x.GetPayload().(*Message_Rbc); ok {
		return x.Rbc
	}
	return nil
}

func (x *Message) GetBba() *BBA {
	if x, ok := x.GetPayload().(*Message_Bba); ok {
		return x.Bba
	}
	return nil
}

func (x *Message) GetCe() *CE {
	if x, ok := x.GetPayload().(*Message_Ce); ok {
		return x.Ce
	}
	return nil
}

func (x *Message) GetDs() *DecShare {
	if x, ok := x.GetPayload().(*Message_Ds); ok {
		return x.Ds
	}
	return nil
}

type isMessage_Payload interface {
	isMessage_Payload()
}

type Message_Rbc struct {
	Rbc *RBC `protobuf:"bytes,6,opt,name=rbc,proto3,oneof"`
}

type Message_Bba struct {
	Bba *BBA `protobuf:"bytes,7,opt,name=bba,proto3,oneof"`
}

type Message_Ce struct {
	Ce *CE `protobuf:"bytes,8,opt,name=ce,proto3,oneof"`
}

type Message_Ds struct {
	Ds *DecShare `protobuf:"bytes,9,opt,name=ds,proto3,oneof"`
}

func (*Message_Rbc) isMessage_Payload() {}

func (*Message_Bba) isMessage_Payload() {}

func (*Message_Ce) isMessage_Payload() {}

func (*Message_Ds) isMessage_Payload() {}

type RBC struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// marshaled data by type
	Payload []byte `protobuf:"bytes,1,opt,name=payload,proto3" json:"payload,omitempty"`
	// The length of original data
	ContentLength uint64 `protobuf:"varint,2,opt,name=contentLength,proto3" json:"contentLength,omitempty"`
	// isIndexRbc
	IsIndexRbc bool     `protobuf:"varint,3,opt,name=isIndexRbc,proto3" json:"isIndexRbc,omitempty"`
	Type       RBC_Type `protobuf:"varint,4,opt,name=type,proto3,enum=pb.RBC_Type" json:"type,omitempty"`
}

func (x *RBC) Reset() {
	*x = RBC{}
	if protoimpl.UnsafeEnabled {
		mi := &file_message_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RBC) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RBC) ProtoMessage() {}

func (x *RBC) ProtoReflect() protoreflect.Message {
	mi := &file_message_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RBC.ProtoReflect.Descriptor instead.
func (*RBC) Descriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{1}
}

func (x *RBC) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

func (x *RBC) GetContentLength() uint64 {
	if x != nil {
		return x.ContentLength
	}
	return 0
}

func (x *RBC) GetIsIndexRbc() bool {
	if x != nil {
		return x.IsIndexRbc
	}
	return false
}

func (x *RBC) GetType() RBC_Type {
	if x != nil {
		return x.Type
	}
	return RBC_VAL
}

type BBA struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// marshaled data by type
	Payload []byte `protobuf:"bytes,1,opt,name=payload,proto3" json:"payload,omitempty"`
	// each epoch's BBA round, it is different with epoch
	Round uint64   `protobuf:"varint,2,opt,name=round,proto3" json:"round,omitempty"`
	Type  BBA_Type `protobuf:"varint,3,opt,name=type,proto3,enum=pb.BBA_Type" json:"type,omitempty"`
}

func (x *BBA) Reset() {
	*x = BBA{}
	if protoimpl.UnsafeEnabled {
		mi := &file_message_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BBA) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BBA) ProtoMessage() {}

func (x *BBA) ProtoReflect() protoreflect.Message {
	mi := &file_message_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BBA.ProtoReflect.Descriptor instead.
func (*BBA) Descriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{2}
}

func (x *BBA) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

func (x *BBA) GetRound() uint64 {
	if x != nil {
		return x.Round
	}
	return 0
}

func (x *BBA) GetType() BBA_Type {
	if x != nil {
		return x.Type
	}
	return BBA_BVAL
}

type CE struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// CE payload
	Payload []byte  `protobuf:"bytes,1,opt,name=payload,proto3" json:"payload,omitempty"`
	Type    CE_Type `protobuf:"varint,2,opt,name=type,proto3,enum=pb.CE_Type" json:"type,omitempty"`
}

func (x *CE) Reset() {
	*x = CE{}
	if protoimpl.UnsafeEnabled {
		mi := &file_message_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CE) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CE) ProtoMessage() {}

func (x *CE) ProtoReflect() protoreflect.Message {
	mi := &file_message_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CE.ProtoReflect.Descriptor instead.
func (*CE) Descriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{3}
}

func (x *CE) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

func (x *CE) GetType() CE_Type {
	if x != nil {
		return x.Type
	}
	return CE_SHARE
}

type DecShare struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Payload []byte `protobuf:"bytes,1,opt,name=payload,proto3" json:"payload,omitempty"`
}

func (x *DecShare) Reset() {
	*x = DecShare{}
	if protoimpl.UnsafeEnabled {
		mi := &file_message_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DecShare) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DecShare) ProtoMessage() {}

func (x *DecShare) ProtoReflect() protoreflect.Message {
	mi := &file_message_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DecShare.ProtoReflect.Descriptor instead.
func (*DecShare) Descriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{4}
}

func (x *DecShare) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

var File_message_proto protoreflect.FileDescriptor

var file_message_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x02, 0x70, 0x62, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0xaa, 0x02, 0x0a, 0x07, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x12, 0x1c, 0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x12, 0x1a,
	0x0a, 0x08, 0x70, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x08, 0x70, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x65, 0x72, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x65,
	0x6e, 0x64, 0x65, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x65, 0x6e, 0x64,
	0x65, 0x72, 0x12, 0x38, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d,
	0x70, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x14, 0x0a, 0x05,
	0x65, 0x70, 0x6f, 0x63, 0x68, 0x18, 0x05, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05, 0x65, 0x70, 0x6f,
	0x63, 0x68, 0x12, 0x1b, 0x0a, 0x03, 0x72, 0x62, 0x63, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x07, 0x2e, 0x70, 0x62, 0x2e, 0x52, 0x42, 0x43, 0x48, 0x00, 0x52, 0x03, 0x72, 0x62, 0x63, 0x12,
	0x1b, 0x0a, 0x03, 0x62, 0x62, 0x61, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x07, 0x2e, 0x70,
	0x62, 0x2e, 0x42, 0x42, 0x41, 0x48, 0x00, 0x52, 0x03, 0x62, 0x62, 0x61, 0x12, 0x18, 0x0a, 0x02,
	0x63, 0x65, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x06, 0x2e, 0x70, 0x62, 0x2e, 0x43, 0x45,
	0x48, 0x00, 0x52, 0x02, 0x63, 0x65, 0x12, 0x1e, 0x0a, 0x02, 0x64, 0x73, 0x18, 0x09, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x0c, 0x2e, 0x70, 0x62, 0x2e, 0x44, 0x65, 0x63, 0x53, 0x68, 0x61, 0x72, 0x65,
	0x48, 0x00, 0x52, 0x02, 0x64, 0x73, 0x42, 0x09, 0x0a, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61,
	0x64, 0x22, 0xad, 0x01, 0x0a, 0x03, 0x52, 0x42, 0x43, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x79,
	0x6c, 0x6f, 0x61, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x70, 0x61, 0x79, 0x6c,
	0x6f, 0x61, 0x64, 0x12, 0x24, 0x0a, 0x0d, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x4c, 0x65,
	0x6e, 0x67, 0x74, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x0d, 0x63, 0x6f, 0x6e, 0x74,
	0x65, 0x6e, 0x74, 0x4c, 0x65, 0x6e, 0x67, 0x74, 0x68, 0x12, 0x1e, 0x0a, 0x0a, 0x69, 0x73, 0x49,
	0x6e, 0x64, 0x65, 0x78, 0x52, 0x62, 0x63, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0a, 0x69,
	0x73, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x52, 0x62, 0x63, 0x12, 0x20, 0x0a, 0x04, 0x74, 0x79, 0x70,
	0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0c, 0x2e, 0x70, 0x62, 0x2e, 0x52, 0x42, 0x43,
	0x2e, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x22, 0x24, 0x0a, 0x04, 0x54,
	0x79, 0x70, 0x65, 0x12, 0x07, 0x0a, 0x03, 0x56, 0x41, 0x4c, 0x10, 0x00, 0x12, 0x08, 0x0a, 0x04,
	0x45, 0x43, 0x48, 0x4f, 0x10, 0x01, 0x12, 0x09, 0x0a, 0x05, 0x52, 0x45, 0x41, 0x44, 0x59, 0x10,
	0x02, 0x22, 0xa5, 0x01, 0x0a, 0x03, 0x42, 0x42, 0x41, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x79,
	0x6c, 0x6f, 0x61, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x70, 0x61, 0x79, 0x6c,
	0x6f, 0x61, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x72, 0x6f, 0x75, 0x6e, 0x64, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x04, 0x52, 0x05, 0x72, 0x6f, 0x75, 0x6e, 0x64, 0x12, 0x20, 0x0a, 0x04, 0x74, 0x79, 0x70,
	0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0c, 0x2e, 0x70, 0x62, 0x2e, 0x42, 0x42, 0x41,
	0x2e, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x22, 0x4c, 0x0a, 0x04, 0x54,
	0x79, 0x70, 0x65, 0x12, 0x08, 0x0a, 0x04, 0x42, 0x56, 0x41, 0x4c, 0x10, 0x00, 0x12, 0x07, 0x0a,
	0x03, 0x41, 0x55, 0x58, 0x10, 0x01, 0x12, 0x0d, 0x0a, 0x09, 0x4d, 0x41, 0x49, 0x4e, 0x5f, 0x56,
	0x4f, 0x54, 0x45, 0x10, 0x02, 0x12, 0x0e, 0x0a, 0x0a, 0x46, 0x49, 0x4e, 0x41, 0x4c, 0x5f, 0x56,
	0x4f, 0x54, 0x45, 0x10, 0x03, 0x12, 0x12, 0x0a, 0x0e, 0x43, 0x4f, 0x49, 0x4e, 0x5f, 0x53, 0x49,
	0x47, 0x5f, 0x53, 0x48, 0x41, 0x52, 0x45, 0x10, 0x04, 0x22, 0x52, 0x0a, 0x02, 0x43, 0x45, 0x12,
	0x18, 0x0a, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c,
	0x52, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x12, 0x1f, 0x0a, 0x04, 0x74, 0x79, 0x70,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0b, 0x2e, 0x70, 0x62, 0x2e, 0x43, 0x45, 0x2e,
	0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x22, 0x11, 0x0a, 0x04, 0x54, 0x79,
	0x70, 0x65, 0x12, 0x09, 0x0a, 0x05, 0x53, 0x48, 0x41, 0x52, 0x45, 0x10, 0x00, 0x22, 0x24, 0x0a,
	0x08, 0x44, 0x65, 0x63, 0x53, 0x68, 0x61, 0x72, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x79,
	0x6c, 0x6f, 0x61, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x70, 0x61, 0x79, 0x6c,
	0x6f, 0x61, 0x64, 0x32, 0x40, 0x0a, 0x0d, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x53, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x12, 0x2f, 0x0a, 0x0d, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x53,
	0x74, 0x72, 0x65, 0x61, 0x6d, 0x12, 0x0b, 0x2e, 0x70, 0x62, 0x2e, 0x4d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x1a, 0x0b, 0x2e, 0x70, 0x62, 0x2e, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x22,
	0x00, 0x28, 0x01, 0x30, 0x01, 0x42, 0x07, 0x5a, 0x05, 0x2e, 0x2f, 0x3b, 0x70, 0x62, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_message_proto_rawDescOnce sync.Once
	file_message_proto_rawDescData = file_message_proto_rawDesc
)

func file_message_proto_rawDescGZIP() []byte {
	file_message_proto_rawDescOnce.Do(func() {
		file_message_proto_rawDescData = protoimpl.X.CompressGZIP(file_message_proto_rawDescData)
	})
	return file_message_proto_rawDescData
}

var file_message_proto_enumTypes = make([]protoimpl.EnumInfo, 3)
var file_message_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_message_proto_goTypes = []interface{}{
	(RBC_Type)(0),                 // 0: pb.RBC.Type
	(BBA_Type)(0),                 // 1: pb.BBA.Type
	(CE_Type)(0),                  // 2: pb.CE.Type
	(*Message)(nil),               // 3: pb.Message
	(*RBC)(nil),                   // 4: pb.RBC
	(*BBA)(nil),                   // 5: pb.BBA
	(*CE)(nil),                    // 6: pb.CE
	(*DecShare)(nil),              // 7: pb.DecShare
	(*timestamppb.Timestamp)(nil), // 8: google.protobuf.Timestamp
}
var file_message_proto_depIdxs = []int32{
	8, // 0: pb.Message.timestamp:type_name -> google.protobuf.Timestamp
	4, // 1: pb.Message.rbc:type_name -> pb.RBC
	5, // 2: pb.Message.bba:type_name -> pb.BBA
	6, // 3: pb.Message.ce:type_name -> pb.CE
	7, // 4: pb.Message.ds:type_name -> pb.DecShare
	0, // 5: pb.RBC.type:type_name -> pb.RBC.Type
	1, // 6: pb.BBA.type:type_name -> pb.BBA.Type
	2, // 7: pb.CE.type:type_name -> pb.CE.Type
	3, // 8: pb.StreamService.MessageStream:input_type -> pb.Message
	3, // 9: pb.StreamService.MessageStream:output_type -> pb.Message
	9, // [9:10] is the sub-list for method output_type
	8, // [8:9] is the sub-list for method input_type
	8, // [8:8] is the sub-list for extension type_name
	8, // [8:8] is the sub-list for extension extendee
	0, // [0:8] is the sub-list for field type_name
}

func init() { file_message_proto_init() }
func file_message_proto_init() {
	if File_message_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_message_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Message); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_message_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RBC); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_message_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BBA); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_message_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CE); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_message_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DecShare); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_message_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*Message_Rbc)(nil),
		(*Message_Bba)(nil),
		(*Message_Ce)(nil),
		(*Message_Ds)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_message_proto_rawDesc,
			NumEnums:      3,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_message_proto_goTypes,
		DependencyIndexes: file_message_proto_depIdxs,
		EnumInfos:         file_message_proto_enumTypes,
		MessageInfos:      file_message_proto_msgTypes,
	}.Build()
	File_message_proto = out.File
	file_message_proto_rawDesc = nil
	file_message_proto_goTypes = nil
	file_message_proto_depIdxs = nil
}