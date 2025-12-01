package cfg

import (
	"github.com/Nash0810/BPF-Insight/pkg/parser"
)

// ============================================================
// Basic Block Structure
// ============================================================

type BasicBlock struct {
	ID           int
	StartOffset  int // byte offset
	EndOffset    int // byte offset
	Instructions []parser.Instruction
	Successors   []*BasicBlock
	Predecessors []*BasicBlock
}

// ============================================================
// Edge Types
// ============================================================

type EdgeType string

const (
	EdgeFallthrough EdgeType = "fallthrough"
	EdgeJump        EdgeType = "jump"
	EdgeBackEdge    EdgeType = "back-edge"
)

type Edge struct {
	From *BasicBlock
	To   *BasicBlock
	Type EdgeType
}

type CFG struct {
	Blocks    []*BasicBlock
	Entry     *BasicBlock
	Edges     []Edge
	BackEdges []Edge
}

type LoopInfo struct {
	Header int
	Blocks []int
}

// ============================================================
// Utility: ensure successor lists do not duplicate blocks
// ============================================================

func addUniqueBlock(list []*BasicBlock, v *BasicBlock) []*BasicBlock {
	for _, x := range list {
		if x.ID == v.ID {
			return list
		}
	}
	return append(list, v)
}

// ============================================================
// Basic Block Construction (Leader-based)
// ============================================================

func BuildBasicBlocks(insns []parser.Instruction) []*BasicBlock {
	leaders := map[int]bool{0: true}

	// Find block leaders using classic rules
	for i, ins := range insns {

		// Identify jump instructions
		if isJump(ins) {
			offset := int(ins.OffsetVal)

			target := i + 1 + offset
			if target >= 0 && target < len(insns) {
				leaders[target] = true
			}

			// fall-through
			if i+1 < len(insns) {
				leaders[i+1] = true
			}
		}
	}

	// Build blocks
	var blocks []*BasicBlock
	cur := &BasicBlock{ID: 0, StartOffset: 0}

	for i, ins := range insns {
		if leaders[i] && len(cur.Instructions) > 0 {
			cur.EndOffset = i * 8
			blocks = append(blocks, cur)

			cur = &BasicBlock{
				ID:          len(blocks),
				StartOffset: i * 8,
			}
		}
		cur.Instructions = append(cur.Instructions, ins)
	}

	// Final block
	if len(cur.Instructions) > 0 {
		cur.ID = len(blocks)
		cur.EndOffset = len(insns) * 8
		blocks = append(blocks, cur)
	}

	return blocks
}

// ============================================================
// Helper: identify jumps & exit instructions
// ============================================================

func isJump(ins parser.Instruction) bool {
	return ins.IsJump()
}

func isExit(ins parser.Instruction) bool {
	return ins.IsExit()
}

// ============================================================
// CFG Construction
// ============================================================

func BuildCFG(blocks []*BasicBlock) *CFG {
	if len(blocks) == 0 {
		return &CFG{Blocks: blocks}
	}

	cfg := &CFG{
		Blocks: blocks,
		Entry:  blocks[0],
		Edges:  []Edge{},
	}

	for i, blk := range blocks {
		if len(blk.Instructions) == 0 {
			continue
		}

		last := blk.Instructions[len(blk.Instructions)-1]
		insnIdx := blk.StartOffset/8 + len(blk.Instructions) - 1

		// Jump instruction
		if isJump(last) && !isExit(last) {

			jumpTargetIdx := insnIdx + 1 + int(last.OffsetVal)
			jumpTargetOffset := jumpTargetIdx * 8

			// Find jump target block
			var targetBlock *BasicBlock
			for _, b := range blocks {
				if jumpTargetOffset >= b.StartOffset && jumpTargetOffset < b.EndOffset {
					targetBlock = b
					break
				}
			}

			if targetBlock != nil {
				edge := Edge{From: blk, To: targetBlock, Type: EdgeJump}
				cfg.Edges = append(cfg.Edges, edge)
				blk.Successors = addUniqueBlock(blk.Successors, targetBlock)
				targetBlock.Predecessors = addUniqueBlock(targetBlock.Predecessors, blk)
			}

			// Fall-through edge
			if i+1 < len(blocks) {
				edge := Edge{From: blk, To: blocks[i+1], Type: EdgeFallthrough}
				cfg.Edges = append(cfg.Edges, edge)
				blk.Successors = addUniqueBlock(blk.Successors, blocks[i+1])
				blocks[i+1].Predecessors = addUniqueBlock(blocks[i+1].Predecessors, blk)
			}

			continue
		}

		// Normal fall-through (not exit)
		if !isExit(last) && i+1 < len(blocks) {
			edge := Edge{From: blk, To: blocks[i+1], Type: EdgeFallthrough}
			cfg.Edges = append(cfg.Edges, edge)
			blk.Successors = addUniqueBlock(blk.Successors, blocks[i+1])
			blocks[i+1].Predecessors = addUniqueBlock(blocks[i+1].Predecessors, blk)
		}
	}

	cfg.BackEdges = detectBackEdges(cfg)
	return cfg
}

// ============================================================
// Back-edge (loop) detection (DFS)
// ============================================================

func detectBackEdges(cfg *CFG) []Edge {
	var backEdges []Edge

	visited := make(map[int]bool)
	recStack := make(map[int]bool)

	var dfs func(b *BasicBlock)
	dfs = func(b *BasicBlock) {
		visited[b.ID] = true
		recStack[b.ID] = true

		for _, succ := range b.Successors {
			if recStack[succ.ID] {
				// Found a loop (back-edge)
				for _, edge := range cfg.Edges {
					if edge.From == b && edge.To == succ {
						e := edge
						e.Type = EdgeBackEdge
						backEdges = append(backEdges, e)
						break
					}
				}
			} else if !visited[succ.ID] {
				dfs(succ)
			}
		}

		recStack[b.ID] = false
	}

	if cfg.Entry != nil {
		dfs(cfg.Entry)
	}

	return backEdges
}

// ============================================================
// LoopInfo (Legacy Compatibility Layer)
// ============================================================

func DetectLoops(cfg *CFG) []LoopInfo {
	var loops []LoopInfo

	for _, e := range cfg.BackEdges {
		loops = append(loops, LoopInfo{
			Header: e.To.ID,
			Blocks: []int{e.From.ID, e.To.ID},
		})
	}

	return loops
}
