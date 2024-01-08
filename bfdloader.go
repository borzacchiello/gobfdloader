package gobfdloader

// #cgo LDFLAGS: -lbfdloader
//
// #include <bfdloader.h>
// #include <stdlib.h>
import "C"
import (
	"fmt"
	"os"
	"unsafe"
)

const (
	SYMBOL_UNKNOWN  = 0
	SYMBOL_FUNCTION = 1
	SYMBOL_LOCAL    = 2
	SYMBOL_GLOBAL   = 3

	RELOC_UNKNOWN  = 0
	RELOC_FUNCTION = 1
	RELOC_DATA     = 2

	PERM_READ  = 1
	PERM_WRITE = 2
	PERM_EXEC  = 4
)

type Section struct {
	Address    uint64
	Name       string
	Data       []byte
	Permission uint8
}

type Symbol struct {
	Name    string
	Address uint64
	Type    int
}

type Relocation struct {
	Name    string
	Address uint64
	Type    int
}

type LoadedBinary struct {
	Arch       string
	BinaryType string
	Entrypoint uint64

	Sections    []Section
	Symbols     []Symbol
	Relocations []Relocation
}

func (lb *LoadedBinary) String() string {
	return fmt.Sprintf("LoadedBinary { %s | %s | %d sections | %d symbols | %d relocations }",
		lb.BinaryType, lb.Arch, len(lb.Sections), len(lb.Symbols), len(lb.Relocations))
}

func LoadBinary(path string) (*LoadedBinary, error) {
	ctx := &C.LoadedBinary{}
	err := C.bfdloader_load(ctx, C.CString(path))
	if err != 0 {
		return nil, fmt.Errorf("unable to load [errcode: %d]", err)
	}
	defer C.bfdloader_destroy(ctx)

	res := &LoadedBinary{
		Arch:        C.GoString(ctx.arch),
		BinaryType:  C.GoString(ctx.bintype),
		Entrypoint:  uint64(ctx.entrypoint),
		Sections:    make([]Section, 0),
		Symbols:     make([]Symbol, 0),
		Relocations: make([]Relocation, 0),
	}
	nSections := ctx.n_sections
	for i := 0; i < int(nSections); i++ {
		section := (*C.Section)(unsafe.Add(unsafe.Pointer(ctx.sections), unsafe.Sizeof(*ctx.sections)*uintptr(i)))
		res.Sections = append(res.Sections, Section{
			Address:    uint64(section.addr),
			Name:       C.GoString(section.name),
			Data:       C.GoBytes(unsafe.Pointer(section.data), C.int(section.size)),
			Permission: uint8(section.perm),
		})
	}
	nSymbols := ctx.n_symbols
	for i := 0; i < int(nSymbols); i++ {
		symbol := (*C.Symbol)(unsafe.Add(unsafe.Pointer(ctx.symbols), unsafe.Sizeof(*ctx.symbols)*uintptr(i)))
		res.Symbols = append(res.Symbols, Symbol{
			Address: uint64(symbol.addr),
			Name:    C.GoString(symbol.name),
			Type:    int(symbol.ty),
		})
	}
	nRelocations := ctx.n_relocations
	for i := 0; i < int(nRelocations); i++ {
		relocation := (*C.Relocation)(unsafe.Add(unsafe.Pointer(ctx.relocations), unsafe.Sizeof(*ctx.relocations)*uintptr(i)))
		res.Relocations = append(res.Relocations, Relocation{
			Address: uint64(relocation.addr),
			Name:    C.GoString(relocation.name),
			Type:    int(relocation.ty),
		})
	}
	return res, nil
}

func LoadBinaryBlob(data []byte) (*LoadedBinary, error) {
	f, err := os.CreateTemp("", "tmpfile-")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	defer os.Remove(f.Name())

	if _, err = f.Write(data); err != nil {
		return nil, err
	}
	return LoadBinary(f.Name())
}
