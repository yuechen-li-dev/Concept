package concept

// SPIR-V reflection and Concept binding generation for `concept vulkan-bind`.
//
// A compute kernel's interface (entry point, workgroup size, set-0 buffer
// bindings with their access, and the push-constant block) is read from the
// SPIR-V module itself, so it works for any front end that emits standard
// decorations (GLSL through glslang, HLSL and SDSL-V through DXC). The
// generated Concept module is ordinary source: a loader, a push-constant
// layout whose offsets the compiler checks, and a typed Record function whose
// buffer access comes from the shader, which is what the Vulkan library's
// Recording derives barriers from.

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

type SPIRVInterface struct {
	Entry     string
	LocalSize [3]uint32
	Bindings  []SPIRVBinding
	Push      *SPIRVPushBlock
}

type SPIRVBinding struct {
	Binding  uint32
	Kind     string // "storage" or "uniform"
	ReadOnly bool
	Element  string // Concept element type, or "" when the block is not one runtime array of scalars
	Name     string
}

type SPIRVPushBlock struct {
	Size    uint32
	Members []SPIRVPushMember
}

type SPIRVPushMember struct {
	Name   string
	Offset uint32
	Type   string // Concept scalar type
	Count  uint32 // 1 for a scalar, N for a vector or fixed array of N scalars
	Bytes  uint32
}

const (
	spvOpName             = 5
	spvOpMemberName       = 6
	spvOpEntryPoint       = 15
	spvOpExecutionMode    = 16
	spvOpTypeBool         = 20
	spvOpTypeInt          = 21
	spvOpTypeFloat        = 22
	spvOpTypeVector       = 23
	spvOpTypeArray        = 28
	spvOpTypeRuntimeArray = 29
	spvOpTypeStruct       = 30
	spvOpTypePointer      = 32
	spvOpConstant         = 43
	spvOpVariable         = 59
	spvOpDecorate         = 71
	spvOpMemberDecorate   = 72
	spvOpExecutionModeId  = 331

	spvDecorationBlock         = 2
	spvDecorationBufferBlock   = 3
	spvDecorationNonWritable   = 24
	spvDecorationBinding       = 33
	spvDecorationDescriptorSet = 34
	spvDecorationOffset        = 35

	spvStorageUniform       = 2
	spvStoragePushConstant  = 9
	spvStorageStorageBuffer = 12

	spvExecutionModelGLCompute  = 5
	spvExecutionModeLocalSize   = 17
	spvExecutionModeLocalSizeId = 38
)

type spvType struct {
	op      uint32
	width   uint32
	signed  bool
	elem    uint32
	length  uint32 // constant id for arrays, component count for vectors
	members []uint32
	storage uint32 // pointer storage class
}

// ReflectSPIRV reads the compute interface of a SPIR-V module.
func ReflectSPIRV(module []byte) (SPIRVInterface, error) {
	var out SPIRVInterface
	if len(module) < 20 || len(module)%4 != 0 {
		return out, fmt.Errorf("not a SPIR-V module: %d bytes", len(module))
	}
	words := make([]uint32, len(module)/4)
	for i := range words {
		words[i] = binary.LittleEndian.Uint32(module[i*4:])
	}
	if words[0] != 0x07230203 {
		return out, fmt.Errorf("not a SPIR-V module: bad magic 0x%08x", words[0])
	}
	names := map[uint32]string{}
	memberNames := map[[2]uint32]string{}
	decorations := map[uint32]map[uint32][]uint32{}
	memberDecorations := map[[2]uint32]map[uint32][]uint32{}
	types := map[uint32]spvType{}
	constants := map[uint32]uint32{}
	variables := map[uint32][2]uint32{} // id -> {pointer type, storage class}
	var entry uint32
	var localSizeIDs []uint32
	for at := 5; at < len(words); {
		count := int(words[at] >> 16)
		op := words[at] & 0xffff
		if count == 0 || at+count > len(words) {
			return out, fmt.Errorf("malformed SPIR-V instruction at word %d", at)
		}
		operands := words[at+1 : at+count]
		switch op {
		case spvOpName:
			names[operands[0]] = spvString(operands[1:])
		case spvOpMemberName:
			memberNames[[2]uint32{operands[0], operands[1]}] = spvString(operands[2:])
		case spvOpEntryPoint:
			if operands[0] == spvExecutionModelGLCompute && out.Entry == "" {
				entry = operands[1]
				out.Entry = spvString(operands[2:])
			}
		case spvOpExecutionMode:
			if operands[0] == entry && operands[1] == spvExecutionModeLocalSize && len(operands) >= 5 {
				out.LocalSize = [3]uint32{operands[2], operands[3], operands[4]}
			}
		case spvOpExecutionModeId:
			if operands[0] == entry && operands[1] == spvExecutionModeLocalSizeId && len(operands) >= 5 {
				localSizeIDs = []uint32{operands[2], operands[3], operands[4]}
			}
		case spvOpDecorate:
			if decorations[operands[0]] == nil {
				decorations[operands[0]] = map[uint32][]uint32{}
			}
			decorations[operands[0]][operands[1]] = operands[2:]
		case spvOpMemberDecorate:
			key := [2]uint32{operands[0], operands[1]}
			if memberDecorations[key] == nil {
				memberDecorations[key] = map[uint32][]uint32{}
			}
			memberDecorations[key][operands[2]] = operands[3:]
		case spvOpTypeBool:
			types[operands[0]] = spvType{op: op}
		case spvOpTypeInt:
			types[operands[0]] = spvType{op: op, width: operands[1], signed: operands[2] != 0}
		case spvOpTypeFloat:
			types[operands[0]] = spvType{op: op, width: operands[1]}
		case spvOpTypeVector:
			types[operands[0]] = spvType{op: op, elem: operands[1], length: operands[2]}
		case spvOpTypeArray:
			types[operands[0]] = spvType{op: op, elem: operands[1], length: operands[2]}
		case spvOpTypeRuntimeArray:
			types[operands[0]] = spvType{op: op, elem: operands[1]}
		case spvOpTypeStruct:
			types[operands[0]] = spvType{op: op, members: append([]uint32{}, operands[1:]...)}
		case spvOpTypePointer:
			types[operands[0]] = spvType{op: op, storage: operands[1], elem: operands[2]}
		case spvOpConstant:
			if len(operands) >= 3 {
				constants[operands[1]] = operands[2]
			}
		case spvOpVariable:
			variables[operands[1]] = [2]uint32{operands[0], operands[2]}
		}
		at += count
	}
	if out.Entry == "" {
		return out, fmt.Errorf("no GLCompute entry point")
	}
	if localSizeIDs != nil {
		for i, id := range localSizeIDs {
			value, ok := constants[id]
			if !ok {
				return out, fmt.Errorf("workgroup size uses a specialization or unknown constant")
			}
			out.LocalSize[i] = value
		}
	}
	if out.LocalSize[0] == 0 {
		return out, fmt.Errorf("the entry point declares no workgroup size")
	}
	hasDecoration := func(id, decoration uint32) bool {
		_, ok := decorations[id][decoration]
		return ok
	}
	ids := make([]uint32, 0, len(variables))
	for id := range variables {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		pointer := types[variables[id][0]]
		storage := variables[id][1]
		block := pointer.elem
		switch storage {
		case spvStoragePushConstant:
			push, err := spvReflectPush(block, types, constants, memberNames, memberDecorations)
			if err != nil {
				return out, err
			}
			out.Push = push
		case spvStorageStorageBuffer, spvStorageUniform:
			binding, ok := decorations[id][spvDecorationBinding]
			if !ok {
				continue
			}
			if set := decorations[id][spvDecorationDescriptorSet]; len(set) == 1 && set[0] != 0 {
				return out, fmt.Errorf("binding %s uses descriptor set %d; only set 0 is supported", names[id], set[0])
			}
			kind := "storage"
			if storage == spvStorageUniform && !hasDecoration(block, spvDecorationBufferBlock) {
				kind = "uniform"
			}
			blockType := types[block]
			readOnly := kind == "uniform" || hasDecoration(id, spvDecorationNonWritable)
			if !readOnly && len(blockType.members) > 0 {
				readOnly = true
				for m := range blockType.members {
					if _, ok := memberDecorations[[2]uint32{block, uint32(m)}][spvDecorationNonWritable]; !ok {
						readOnly = false
					}
				}
			}
			element := ""
			if len(blockType.members) == 1 {
				member := types[blockType.members[0]]
				if member.op == spvOpTypeRuntimeArray {
					element = spvScalarName(types[member.elem])
				}
			}
			name := names[id]
			if name == "" {
				name = names[block]
			}
			out.Bindings = append(out.Bindings, SPIRVBinding{Binding: binding[0], Kind: kind, ReadOnly: readOnly, Element: element, Name: name})
		}
	}
	sort.Slice(out.Bindings, func(i, j int) bool { return out.Bindings[i].Binding < out.Bindings[j].Binding })
	return out, nil
}

func spvReflectPush(block uint32, types map[uint32]spvType, constants map[uint32]uint32, memberNames map[[2]uint32]string, memberDecorations map[[2]uint32]map[uint32][]uint32) (*SPIRVPushBlock, error) {
	structType := types[block]
	push := &SPIRVPushBlock{}
	for m, memberType := range structType.members {
		key := [2]uint32{block, uint32(m)}
		offset, ok := memberDecorations[key][spvDecorationOffset]
		if !ok {
			return nil, fmt.Errorf("push-constant member %d has no Offset", m)
		}
		t := types[memberType]
		count := uint32(1)
		scalar := t
		if t.op == spvOpTypeVector {
			count = t.length
			scalar = types[t.elem]
		} else if t.op == spvOpTypeArray {
			count = constants[t.length]
			scalar = types[t.elem]
		}
		name := spvScalarName(scalar)
		if name == "" || count == 0 {
			return nil, fmt.Errorf("push-constant member %q has a type Concept cannot represent", memberNames[key])
		}
		bytes := scalar.width / 8 * count
		push.Members = append(push.Members, SPIRVPushMember{Name: memberNames[key], Offset: offset[0], Type: name, Count: count, Bytes: bytes})
		if end := offset[0] + bytes; end > push.Size {
			push.Size = end
		}
	}
	push.Size = (push.Size + 3) &^ 3
	return push, nil
}

func spvString(words []uint32) string {
	var b strings.Builder
	for _, word := range words {
		for i := 0; i < 4; i++ {
			c := byte(word >> (8 * i))
			if c == 0 {
				return b.String()
			}
			b.WriteByte(c)
		}
	}
	return b.String()
}

func spvScalarName(t spvType) string {
	switch {
	case t.op == spvOpTypeInt && t.width == 32 && t.signed:
		return "int"
	case t.op == spvOpTypeInt && t.width == 32:
		return "uint32"
	case t.op == spvOpTypeInt && t.width == 64 && !t.signed:
		return "uint64"
	case t.op == spvOpTypeInt && t.width == 16 && !t.signed:
		return "uint16"
	case t.op == spvOpTypeFloat && t.width == 32:
		return "float"
	case t.op == spvOpTypeFloat && t.width == 64:
		return "double"
	}
	return ""
}

// Fingerprint identifies the interface a generated module was made from:
// entry point, workgroup size, bindings with kind, access and element type,
// and push-constant offsets and types. Resource names are excluded, so a
// rename in the shader does not make the binding stale.
func (i SPIRVInterface) Fingerprint() string {
	var b strings.Builder
	fmt.Fprintf(&b, "entry=%s;local=%d,%d,%d;", i.Entry, i.LocalSize[0], i.LocalSize[1], i.LocalSize[2])
	for _, binding := range i.Bindings {
		fmt.Fprintf(&b, "b%d=%s,%v,%s;", binding.Binding, binding.Kind, binding.ReadOnly, binding.Element)
	}
	if i.Push != nil {
		fmt.Fprintf(&b, "push=%d", i.Push.Size)
		for _, member := range i.Push.Members {
			fmt.Fprintf(&b, ",%d:%s*%d", member.Offset, member.Type, member.Count)
		}
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:8])
}

// Describe renders the reflected interface for people.
func (i SPIRVInterface) Describe() string {
	var b strings.Builder
	fmt.Fprintf(&b, "entry %s, workgroup %d x %d x %d\n", i.Entry, i.LocalSize[0], i.LocalSize[1], i.LocalSize[2])
	for _, binding := range i.Bindings {
		access := "read-write"
		if binding.ReadOnly {
			access = "read-only"
		}
		element := binding.Element
		if element == "" {
			element = "byte (structured block)"
		}
		fmt.Fprintf(&b, "binding %d  %-7s  %-10s  %-7s  %s\n", binding.Binding, binding.Kind, access, element, binding.Name)
	}
	if i.Push != nil {
		fmt.Fprintf(&b, "push constants %d bytes:", i.Push.Size)
		for _, member := range i.Push.Members {
			fmt.Fprintf(&b, " %s@%d", member.Name, member.Offset)
		}
		b.WriteString("\n")
	}
	return b.String()
}

type VulkanBindOptions struct {
	Name       string // kernel name, e.g. Scale
	Module     string // module name, e.g. ScaleKernel
	KernelPath string // path the pipeline loads, relative to the module's directory
	Source     string // for the header comment
}

// VulkanBindFingerprintLine is the header line --check compares.
const VulkanBindFingerprintLine = "// Interface fingerprint: "

// GenerateVulkanBinding emits the Concept module for a reflected kernel.
func GenerateVulkanBinding(i SPIRVInterface, options VulkanBindOptions) (string, error) {
	if len(i.Bindings) == 0 {
		return "", fmt.Errorf("the kernel has no buffer bindings")
	}
	if len(i.Bindings) > 16 {
		return "", fmt.Errorf("the kernel has %d bindings; the library binds at most 16", len(i.Bindings))
	}
	name := options.Name
	reserved := map[string]bool{"recording": true, "kernel": true, "push": true, "groupsX": true, "groupsY": true, "groupsZ": true, "bindings": true, "raw": true, "constants": true}
	params := make([]string, len(i.Bindings))
	used := map[string]bool{}
	for index, binding := range i.Bindings {
		param := vulkanBindIdentifier(binding.Name)
		if param == "" {
			param = fmt.Sprintf("binding%d", binding.Binding)
		}
		if reserved[param] || evt1IsReservedWord(param) {
			param += "Buffer"
		}
		for used[param] {
			param += fmt.Sprint(binding.Binding)
		}
		used[param] = true
		params[index] = param
	}
	var b strings.Builder
	fmt.Fprintf(&b, "// Generated by `concept vulkan-bind` from %s. DO NOT EDIT.\n", options.Source)
	fmt.Fprintf(&b, "%s%s\n", VulkanBindFingerprintLine, i.Fingerprint())
	fmt.Fprintf(&b, "module %s;\nprofile Vulkan;\n\n", options.Module)
	b.WriteString("// Reflected interface:\n")
	for _, line := range strings.Split(strings.TrimRight(i.Describe(), "\n"), "\n") {
		fmt.Fprintf(&b, "//   %s\n", line)
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "comptime uint32 %sWorkgroupX = %d;\n", name, i.LocalSize[0])
	fmt.Fprintf(&b, "comptime uint32 %sWorkgroupY = %d;\n", name, i.LocalSize[1])
	fmt.Fprintf(&b, "comptime uint32 %sWorkgroupZ = %d;\n\n", name, i.LocalSize[2])
	if i.Push != nil {
		fmt.Fprintf(&b, "// The push-constant block at the offsets the shader declares.\nlayout %sConstants\n{\n", name)
		for _, member := range i.Push.Members {
			fmt.Fprintf(&b, "    at(%d) %s %s;\n", member.Offset, vulkanBindMemberType(member), vulkanBindIdentifier(member.Name))
		}
		b.WriteString("}\n\n")
		fmt.Fprintf(&b, "static_assert(LayoutSize<%sConstants>() == %d, \"the layout matches the reflected push-constant block\");\n\n", name, i.Push.Size)
		fmt.Fprintf(&b, "// The values %s's push constants carry.\nrecord struct %sPush\n{\n", name, name)
		for _, member := range i.Push.Members {
			fmt.Fprintf(&b, "    %s %s;\n", vulkanBindMemberType(member), vulkanBindIdentifier(member.Name))
		}
		b.WriteString("};\n\n")
	}
	fmt.Fprintf(&b, "// The %s pipeline. Dropping it releases the pipeline.\nclass %s\n{\npublic:\n    owned Pipeline pipeline;\n};\n\n", name, name)
	fmt.Fprintf(&b, "Result<%s, VulkanError> Load%s(ref const Context context)\n{\n", name, name)
	fmt.Fprintf(&b, "    BindingSlot<array>[%d] slots = [", len(i.Bindings))
	for index, binding := range i.Bindings {
		if index > 0 {
			b.WriteString(", ")
		}
		kind := 0
		if binding.Kind == "uniform" {
			kind = 1
		}
		fmt.Fprintf(&b, "BindingSlot{%d, %d}", binding.Binding, kind)
	}
	b.WriteString("];\n")
	pushBytes := uint32(0)
	if i.Push != nil {
		pushBytes = i.Push.Size
	}
	fmt.Fprintf(&b, "    owned Pipeline pipeline = CreatePipeline(ref const context, %q, %q, ReadOnlySpan(slots), %d)?;\n", options.KernelPath, i.Entry, pushBytes)
	fmt.Fprintf(&b, "    owned %s kernel = %s{move pipeline};\n    return Result::Ok(move kernel);\n}\n\n", name, name)
	fmt.Fprintf(&b, "// Workgroups needed to cover items along x.\nuint32 %sGroupsFor(uint32 items)\n{\n    return (items + %d) / %d;\n}\n\n", name, i.LocalSize[0]-1, i.LocalSize[0])
	b.WriteString("// Records one dispatch. Each buffer's access comes from the shader:\n")
	for index, binding := range i.Bindings {
		access := "written"
		if binding.ReadOnly {
			access = "read"
		}
		fmt.Fprintf(&b, "//   %s (binding %d) is %s.\n", params[index], binding.Binding, access)
	}
	b.WriteString("// The Recording derives the barriers from that.\n")
	fmt.Fprintf(&b, "void Record%s(ref Recording recording, ref const %s kernel", name, name)
	for index, binding := range i.Bindings {
		element := binding.Element
		if element == "" {
			element = "byte"
		}
		ownership := "ref"
		if binding.ReadOnly {
			ownership = "ref const"
		}
		fmt.Fprintf(&b, ", %s Buffer<%s> %s", ownership, element, params[index])
	}
	if i.Push != nil {
		fmt.Fprintf(&b, ", %sPush push", name)
	}
	b.WriteString(", uint32 groupsX, uint32 groupsY, uint32 groupsZ)\n{\n")
	fmt.Fprintf(&b, "    Binding<array>[%d] bindings = [", len(i.Bindings))
	for index, binding := range i.Bindings {
		if index > 0 {
			b.WriteString(", ")
		}
		switch {
		case binding.Kind == "uniform":
			fmt.Fprintf(&b, "BindUniform(%d, ref const %s)", binding.Binding, params[index])
		case binding.ReadOnly:
			fmt.Fprintf(&b, "Bind(%d, ref const %s, Access::Read)", binding.Binding, params[index])
		default:
			fmt.Fprintf(&b, "Bind(%d, ref const %s, Access::Write)", binding.Binding, params[index])
		}
	}
	b.WriteString("];\n")
	if i.Push != nil {
		fmt.Fprintf(&b, "    byte<array>[%d] raw = [0...];\n    ref %sConstants constants = bind raw;\n", i.Push.Size, name)
		for _, member := range i.Push.Members {
			field := vulkanBindIdentifier(member.Name)
			fmt.Fprintf(&b, "    constants.%s = push.%s;\n", field, field)
		}
		b.WriteString("    DispatchWith(ref recording, ref const kernel.pipeline, ReadOnlySpan(bindings), ReadOnlySpan(raw), groupsX, groupsY, groupsZ);\n")
	} else {
		b.WriteString("    Dispatch(ref recording, ref const kernel.pipeline, ReadOnlySpan(bindings), groupsX, groupsY, groupsZ);\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}

func vulkanBindMemberType(member SPIRVPushMember) string {
	if member.Count == 1 {
		return member.Type
	}
	return fmt.Sprintf("%s<array>[%d]", member.Type, member.Count)
}

// vulkanBindIdentifier turns a shader name into a lowerCamel Concept name.
func vulkanBindIdentifier(name string) string {
	var parts []string
	var current strings.Builder
	for _, r := range name {
		if r == '_' || r == '.' || r == '-' || !(unicode.IsLetter(r) || unicode.IsDigit(r)) {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	if len(parts) == 0 {
		return ""
	}
	var b strings.Builder
	for index, part := range parts {
		runes := []rune(part)
		if index == 0 {
			runes[0] = unicode.ToLower(runes[0])
		} else {
			runes[0] = unicode.ToUpper(runes[0])
		}
		b.WriteString(string(runes))
	}
	out := b.String()
	if unicode.IsDigit([]rune(out)[0]) {
		out = "v" + out
	}
	return out
}

// VulkanBindName derives the kernel name from a SPIR-V path: scale.spv ->
// Scale, fused_qkv.spv -> FusedQkv.
func VulkanBindName(kernelPath string) string {
	stem := strings.TrimSuffix(path.Base(strings.ReplaceAll(kernelPath, "\\", "/")), path.Ext(kernelPath))
	identifier := vulkanBindIdentifier(stem)
	if identifier == "" {
		return "Kernel"
	}
	runes := []rune(identifier)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// Words a generated parameter name must not be. Concept keywords are
// contextual, so this is conservative: a collision only adds a suffix.
var vulkanBindReserved = map[string]bool{
	"align": true, "and": true, "as": true, "assert": true, "async": true, "at": true, "automata": true, "await": true,
	"bind": true, "bits": true, "borrow": true, "class": true, "comptime": true, "concept": true, "const": true,
	"decide": true, "discard": true, "dyn": true, "else": true, "enum": true, "except": true, "extern": true,
	"false": true, "fn": true, "for": true, "handle": true, "if": true, "import": true, "in": true, "infer": true,
	"input": true, "interface": true, "is": true, "layout": true, "let": true, "machine": true, "match": true,
	"module": true, "move": true, "namespace": true, "not": true, "on": true, "or": true, "otherwise": true,
	"out": true, "over": true, "owned": true, "pop": true, "private": true, "profile": true, "public": true,
	"record": true, "ref": true, "requires": true, "return": true, "scoped": true, "state": true,
	"static_assert": true, "stream": true, "struct": true, "table": true, "template": true, "terminal": true,
	"transition": true, "true": true, "try": true, "typename": true, "unsafe": true, "var": true, "when": true,
	"while": true, "with": true, "yield": true,
}

func evt1IsReservedWord(word string) bool { return vulkanBindReserved[word] }

const vulkanBindHeaderPrefix = "// Generated by `concept vulkan-bind` from "

// evt1CheckVulkanKernelBindings compares every generated kernel module in dir
// with its SPIR-V, when the SPIR-V is present (a GPU-free checkout may not
// have compiled kernels). A module generated from a different interface
// would bind buffers or push constants the kernel no longer has.
func evt1CheckVulkanKernelBindings(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".concept") {
			continue
		}
		modulePath := filepath.Join(dir, entry.Name())
		text, err := os.ReadFile(modulePath)
		if err != nil {
			continue
		}
		lines := strings.SplitN(strings.ReplaceAll(string(text), "\r\n", "\n"), "\n", 3)
		if len(lines) < 2 || !strings.HasPrefix(lines[0], vulkanBindHeaderPrefix) || !strings.HasPrefix(lines[1], VulkanBindFingerprintLine) {
			continue
		}
		source := strings.TrimSuffix(strings.TrimPrefix(lines[0], vulkanBindHeaderPrefix), ". DO NOT EDIT.")
		kernel := filepath.Join(dir, filepath.FromSlash(source))
		spirv, err := os.ReadFile(kernel)
		if err != nil {
			continue
		}
		reflected, err := ReflectSPIRV(spirv)
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.ToSlash(kernel), err)
		}
		recorded := strings.TrimPrefix(lines[1], VulkanBindFingerprintLine)
		if recorded != reflected.Fingerprint() {
			return fmt.Errorf("%s was generated from a different interface than %s now has (%s, was %s); rerun: concept vulkan-bind %s -o %s",
				filepath.ToSlash(modulePath), filepath.ToSlash(kernel), reflected.Fingerprint(), recorded, filepath.ToSlash(kernel), filepath.ToSlash(modulePath))
		}
	}
	return nil
}
