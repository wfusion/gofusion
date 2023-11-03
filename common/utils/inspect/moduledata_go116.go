//go:build go1.16 && !go1.18
// +build go1.16,!go1.18

package inspect

import "unsafe"

type _func struct {
	entry       uintptr // start pc
	nameoff     int32   // function name
	args        int32   // in/out args size
	deferreturn uint32  // offset of start of a deferreturn call instruction from entry, if any.
	pcsp        uint32
	pcfile      uint32
	pcln        uint32
	npcdata     uint32
	cuOffset    uint32  // runtime.cutab offset of this function's CU
	funcID      uint8   // set for certain special runtime functions
	_           [2]byte // pad
	nfuncdata   uint8   // must be last
	argptrs     uintptr
	localptrs   uintptr
}

type _funcTab struct {
	entry   uintptr
	funcoff uintptr
}

type _pcHeader struct {
	magic          uint32  // 0xFFFFFFFA
	pad1, pad2     uint8   // 0,0
	minLC          uint8   // min instruction size
	ptrSize        uint8   // size of a ptr in bytes
	nfunc          int     // number of functions in the module
	nfiles         uint    // number of entries in the file tab.
	funcnameOffset uintptr // offset to the funcnametab variable from _PCHeader
	cuOffset       uintptr // offset to the cutab variable from _PCHeader
	filetabOffset  uintptr // offset to the filetab variable from _PCHeader
	pctabOffset    uintptr // offset to the pctab varible from _PCHeader
	pclnOffset     uintptr // offset to the pclntab variable from _PCHeader
}

type _bitVector struct {
	n        int32 // # of bits
	bytedata *uint8
}

type _ptabEntry struct {
	name int32
	typ  int32
}

type _textSection struct {
	vaddr    uintptr // prelinked section vaddr
	length   uintptr // section length
	baseaddr uintptr // relocated section address
}

// moduledata records information about the layout of the executable
// image. It is written by the linker. Any changes here must be
// matched changes to the code in cmd/internal/ld/symtab.go:symtab.
// moduledata is stored in statically allocated non-pointer memory;
// none of the pointers here are visible to the garbage collector.
type _moduleData struct {
	pcHeader     *_pcHeader
	funcnametab  []byte
	cutab        []uint32
	filetab      []byte
	pctab        []byte
	pclntable    []_func
	ftab         []_funcTab
	findfunctab  *_findFuncBucket
	minpc, maxpc uintptr

	text, etext           uintptr
	noptrdata, enoptrdata uintptr
	data, edata           uintptr
	bss, ebss             uintptr
	noptrbss, enoptrbss   uintptr
	end, gcdata, gcbss    uintptr
	types, etypes         uintptr

	textsectmap []_textSection
	typelinks   []int32 // offsets from types
	itablinks   []unsafe.Pointer

	ptab []_ptabEntry

	pluginpath string
	pkghashes  []struct{}

	modulename   string
	modulehashes []struct{}

	hasmain uint8 // 1 if module contains the main function, 0 otherwise

	gcdatamask, gcbssmask _bitVector

	typemap map[int32]*rtype // offset to *_rtype in previous module

	bad bool // module failed to load and should be ignored

	next *_moduleData
}

type _findFuncBucket struct {
	idx        uint32
	subbuckets [16]byte
}
