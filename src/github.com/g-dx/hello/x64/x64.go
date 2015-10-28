package x64
import (
    "bytes"
    "encoding/binary"
    "fmt"
)

// Background
//
// http://www.swansontec.com/sregisters.html
// http://eli.thegreenplace.net/2011/09/06/stack-frame-layout-on-x86-64/
// http://unixwiz.net/techtips/win32-callconv-asm.html/

// Tutorial
// http://www.tptp.cc/mirrors/siyobik.info/view/x86-tutorial/index.html

// main registers of x64
const (

    // x86 (previously prefixed with 'e' instead of 'r') general purpose registers
    Rax = iota
    Rcx
    Rdx
    Rbx

    Rsp // Stack Pointer
    Rbp // Base Pointer
    Rsi // Source Index
    Rdi // Destination Index

    // x64 general purpose registers
    R8
    R9
    R10
    R11
    R12
    R13
    R14
    R15
)

//======================================================================================================================

type Opcode interface {
	Len() uint8
	Bytes() []byte
}

type Rva *uint64

//======================================================================================================================

type SimpleOpcode []byte

func (op SimpleOpcode) Len() uint8 {
	return uint8(len(op))
}

func (op SimpleOpcode) Bytes() []byte {
	return []byte(op)
}

// Opcode containing an unknown RVA value
type RvaOpcode struct {
	buf []byte
    Rva Rva
}

func (rp RvaOpcode) Len() uint8 {
	return uint8(len(rp.buf)) + 8 // rva are 64-bit values
}

func (rp RvaOpcode) Bytes() []byte {
	panic("Not implemented")
}

//======================================================================================================================

type OpcodeList struct {
    Ops []Opcode
    rva uint64
}

func NewOpcodeList(rva uint64) *OpcodeList {
    return &OpcodeList{rva : rva}
}

// Return current RVA
func (ol *OpcodeList) Rva() uint64 {
    return ol.rva
}

func (ol *OpcodeList) Add(op Opcode) {
	ol.Ops = append(ol.Ops, op)
	ol.rva = ol.rva + uint64(op.Len())
}

func (ol *OpcodeList) ToBuffer() []byte {
	var buf bytes.Buffer
	for _, op := range ol.Ops {
		buf.Write(op.Bytes())
	}
	return buf.Bytes()
}

// Instructions

// Call function at the supplied relative virtual address (RVA).
func (ol *OpcodeList) CALL(rva uint64) {
	// TODO: Can't write RVA value yet!
    ol.Add(op(0xFF).Bytes(0x14, 0x25).Write(uint32(rva)).Build())
}

// Call function at the supplied relative virtual address (RVA).
func (ol *OpcodeList) CALLI(rva Rva) {
	// TODO: Can't write RVA value yet!
	ol.Add(op(0xFF).Bytes(0x14, 0x25).Write(rva).Build())
}

func (ol *OpcodeList) MOV(srcReg uint8, destReg uint8) {
    // set register direct mode
    mod := uint8(0xC0)
    mod = mod | (srcReg << 3)
    mod = mod | destReg
    ol.Add(op(0x8B).Bytes(mod).Build()) // TODO: Verify! What about r8 - r15?
}

// Move data from memory location to register
func (ol *OpcodeList) MOVM(srcReg int, destReg uint) {
    // What kind of mov?
    // 0x41 is REX prefix required for referencing rd9, rd8 and other registers
	panic("MOVM not implemented")
}

// Move immediate value to register
// TODO:
//    1. Properly handle register addressing
//    2. Update to support 64 bit values
//    3. Double check the representation of negative number is what Microsoft ABI expects
func (ol *OpcodeList) MOVI(destReg int, val int32) {

    code := byte(0x00)
    switch destReg {
        case Rcx: code = 0xB9
        default:
            panic(fmt.Sprintf("MOVI: %x, %x - register not implemented yet.", destReg, val))
    }

	ol.Add(op(code).Write(val).Build())
}

// Push from register to top of stack
func (ol *OpcodeList) PUSH(srcReg byte) {
    ol.Add(op(0x50 + srcReg).Build()) // TODO: Verify! What about r8 - r15?
}

// Push immediate value to top of stack
func (ol *OpcodeList) PUSHI(val uint32) {

    code := uint8(0x68)
    if val <= 255 {
        code = 0x6A
    }

    ol.Add(op(code).Write(val).Build())
}

// Pop from top of stack to register
func (ol *OpcodeList) POP(destReg byte) {
    ol.Add(op(0x58 + destReg).Build()) // TODO: Verify! What about r8 - r15
}

// Return from function
func (ol *OpcodeList) RET() {
    ol.Add(op(0xC3).Build())
}

func (ol *OpcodeList) SUB(reg byte, n uint32) {
    ol.Add(SimpleOpcode{})
}

//======================================================================================================================

type opcodeBuilder struct {
	buf bytes.Buffer
}

func op(opCode uint8) *opcodeBuilder {
	builder := &opcodeBuilder{}
	return builder.Bytes(opCode)
}

func (ob *opcodeBuilder) Bytes(bytes...uint8) *opcodeBuilder {
	ob.Write(bytes)
	return ob
}

func (ob *opcodeBuilder) Build() Opcode {
	return SimpleOpcode(ob.buf.Bytes())
}

func (ob *opcodeBuilder) Write(data interface{}) *opcodeBuilder {
	err := binary.Write(&ob.buf, binary.LittleEndian, data)
	if err != nil {
		panic(err)
	}
	return ob
}