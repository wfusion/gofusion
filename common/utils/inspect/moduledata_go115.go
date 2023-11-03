// +build go1.15,!go1.16

package inspect

type _func struct {
	entry       uintptr // start pc
	nameoff     int32   // function name
	args        int32   // in/out args size
	deferreturn uint32  // offset of start of a deferreturn call instruction from entry, if any.
	pcsp        int32
	pcfile      int32
	pcln        int32
	npcdata     int32
	funcID      uint8   // set for certain special runtime functions
	_           [2]int8 // unused
	nfuncdata   uint8   // must be last
	argptrs     uintptr
	localptrs   uintptr
}

type _funcTab struct {
	entry   uintptr
	funcoff uintptr
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

type _moduleData struct {
	pclntable    []byte
	ftab         []_funcTab
	filetab      []uint32
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
	itablinks   []*itab

	ptab []_ptabEntry

	pluginpath string
	pkghashes  []byte

	modulename   string
	modulehashes []byte

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
