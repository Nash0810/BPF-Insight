package cfg

import (
    "github.com/yourusername/bpfinsight/pkg/parser"
)

type BasicBlock struct {
    ID           int
    Instructions []parser.Instruction
    Successors   []int
    Predecessors []int
}

type CFG struct {
    Blocks []*BasicBlock
}

type LoopInfo struct {
    Header int
    Blocks []int
}

// Avoid duplicate edges
func addUniqueEdge(list []int, v int) []int {
    for _, x := range list {
        if x == v {
            return list
        }
    }
    return append(list, v)
}

// ------------------------------------------------------------
// Build basic blocks using standard leader rules
// ------------------------------------------------------------

func BuildBasicBlocks(insns []parser.Instruction) []*BasicBlock {
    leaders := map[int]bool{0: true}

    for i, ins := range insns {
        // If this is a jump, mark its target as a leader
        if ins.IsJump() {
            target := i + 1 + int(ins.OffsetVal)
            if target >= 0 && target < len(insns) {
                leaders[target] = true
            }

            // Mark fall-through (next instruction)
            if i+1 < len(insns) {
                leaders[i+1] = true
            }
        }
    }

    // Construct blocks
    var blocks []*BasicBlock
    cur := &BasicBlock{ID: 0}

    for i, ins := range insns {
        if leaders[i] && !(cur.ID == 0 && len(cur.Instructions) == 0) {
            blocks = append(blocks, cur)
            cur = &BasicBlock{ID: len(blocks)}
        }
        cur.Instructions = append(cur.Instructions, ins)
    }

    // Add last block
    if len(cur.Instructions) > 0 {
        cur.ID = len(blocks)
        blocks = append(blocks, cur)
    }

    return blocks
}

// ------------------------------------------------------------
// Build CFG from blocks
// ------------------------------------------------------------

func BuildCFG(blocks []*BasicBlock) *CFG {
    cfg := &CFG{Blocks: blocks}

    // Map: instruction index → block ID
    instToBlock := make(map[int]int)
    for _, blk := range blocks {
        for _, ins := range blk.Instructions {
            instToBlock[ins.Offset] = blk.ID
        }
    }

    for i, blk := range blocks {
        last := blk.Instructions[len(blk.Instructions)-1]

        // Jump instruction: add jump edge AND fall-through edge
        if last.IsJump() {
            jumpTarget := last.Offset + 1 + int(last.OffsetVal)
            if bid, ok := instToBlock[jumpTarget]; ok {
                blk.Successors = addUniqueEdge(blk.Successors, bid)
                blocks[bid].Predecessors = addUniqueEdge(blocks[bid].Predecessors, blk.ID)
            }

            if i+1 < len(blocks) {
                blk.Successors = addUniqueEdge(blk.Successors, blocks[i+1].ID)
                blocks[i+1].Predecessors = addUniqueEdge(blocks[i+1].Predecessors, blk.ID)
            }

        } else if !last.IsExit() {
            // Normal fall-through
            if i+1 < len(blocks) {
                blk.Successors = addUniqueEdge(blk.Successors, blocks[i+1].ID)
                blocks[i+1].Predecessors = addUniqueEdge(blocks[i+1].Predecessors, blk.ID)
            }
        }
    }

    return cfg
}

// ------------------------------------------------------------
// Detect loops (simple back-edge detection for now)
// ------------------------------------------------------------

func DetectLoops(cfg *CFG) []LoopInfo {
    var loops []LoopInfo
    for _, blk := range cfg.Blocks {
        for _, succ := range blk.Successors {
            if succ <= blk.ID {
                loops = append(loops, LoopInfo{
                    Header: succ,
                    Blocks: []int{blk.ID, succ},
                })
            }
        }
    }
    return loops
}
