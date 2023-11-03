//go:build go1.21
// +build go1.21

package inspect

import (
	_ "unsafe"
)

// A FuncFlag holds bits about a function.
// This list must match the list in cmd/internal/objabi/funcid.go.
type funcFlag uint8

type _func struct {
	entryOff    uint32 // start pc
	nameoff     int32  // function name
	args        int32  // in/out args size
	deferreturn uint32 // offset of start of a deferreturn call instruction from entry, if any.
	pcsp        uint32
	pcfile      uint32
	pcln        uint32
	npcdata     uint32
	cuOffset    uint32 // runtime.cutab offset of this function's CU
	funcID      uint8  // set for certain special runtime functions
	flag        funcFlag
	_           [1]byte // pad
	nfuncdata   uint8   // must be last
	argptrs     uint32
	localptrs   uint32
}

type _funcTab struct {
	entry   uint32
	funcoff uint32
}

type _pcHeader struct {
	magic          uint32  // 0xFFFFFFF1
	pad1, pad2     uint8   // 0,0
	minLC          uint8   // min instruction size
	ptrSize        uint8   // size of a ptr in bytes
	nfunc          int     // number of functions in the module
	nfiles         uint    // number of entries in the file tab
	textStart      uintptr // base for function entry PC offsets in this module, equal to moduledata.text
	funcnameOffset uintptr // offset to the funcnametab variable from pcHeader
	cuOffset       uintptr // offset to the cutab variable from pcHeader
	filetabOffset  uintptr // offset to the filetab variable from pcHeader
	pctabOffset    uintptr // offset to the pctab variable from pcHeader
	pclnOffset     uintptr // offset to the pclntab variable from pcHeader
}

type _bitVector struct {
	n        int32 // # of bits
	bytedata *uint8
}

type _ptabEntry struct {
	name nameOff
	typ  typeOff
}

type _textSection struct {
	vaddr    uintptr // prelinked section vaddr
	end      uintptr // vaddr + section length
	baseaddr uintptr // relocated section address
}

// An initTask represents the set of initializations that need to be done for a package.
// Keep in sync with ../../test/noinit.go:initTask
type initTask struct {
	state uint32 // 0 = uninitialized, 1 = in progress, 2 = done
	nfns  uint32
	// followed by nfns pcs, uintptr sized, one per init function to run
}

// moduledata records information about the layout of the executable
// image. It is written by the linker. Any changes here must be
// matched changes to the code in cmd/link/internal/ld/symtab.go:symtab.
// moduledata is stored in statically allocated non-pointer memory;
// none of the pointers here are visible to the garbage collector.
type _moduleData struct {
	notInHeap // Only in static data

	pcHeader     *_pcHeader
	funcnametab  []byte
	cutab        []uint32
	filetab      []byte
	pctab        []byte
	pclntable    []byte
	ftab         []_funcTab
	findfunctab  uintptr
	minpc, maxpc uintptr

	text, etext           uintptr
	noptrdata, enoptrdata uintptr
	data, edata           uintptr
	bss, ebss             uintptr
	noptrbss, enoptrbss   uintptr
	covctrs, ecovctrs     uintptr
	end, gcdata, gcbss    uintptr
	types, etypes         uintptr
	rodata                uintptr
	gofunc                uintptr // go.func.*

	textsectmap []_textSection
	typelinks   []int32 // offsets from types
	itablinks   []*itab

	ptab []_ptabEntry

	pluginpath string
	pkghashes  []_modulehash

	// This slice records the initializing tasks that need to be
	// done to start up the program. It is built by the linker.
	inittasks []*initTask

	modulename   string
	modulehashes []_modulehash

	hasmain               uint8 // 1 if module contains the main function, 0 otherwise
	gcdatamask, gcbssmask _bitVector

	typemap map[typeOff]*rtype // offset to *_rtype in previous module

	bad bool // module failed to load and should be ignored

	next *_moduleData
}

type _modulehash struct {
	modulename   string
	linktimehash string
	runtimehash  *string
}

// nih runtime/internal/sys.nih
// NOTE: keep in sync with cmd/compile/internal/types.CalcSize
// to make the compiler recognize this as an intrinsic type.
type nih struct{}

// notInHeap runtime/internal/sys.NotInHeap
// is a type must never be allocated from the GC'd heap or on the stack,
// and is called not-in-heap.
type notInHeap struct{ _ nih }

//go:linkname textAddr runtime.(*moduledata).textAddr
func textAddr(md *_moduleData, off32 uint32) uintptr
