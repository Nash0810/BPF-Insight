package cfg

import (
    "github.com/Nash0810/BPF-Insight/pkg/parser"
)

type BasicBlock struct {
    ID           int
    StartOffset  int  // Byte offset where block starts
    EndOffset    int  // Byte offset where block ends
    Instructions []parser.Instruction
    Successors   []*BasicBlock
    Predecessors []*BasicBlock
}

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

// Avoid duplicate edges (for pointer-based successors/predecessors)
func addUniqueBlock(list []*BasicBlock, v *BasicBlock) []*BasicBlock {
    for _, x := range list {
        if x.ID == v.ID {
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

    // Construct blocks with proper offsets
    var blocks []*BasicBlock
    cur := &BasicBlock{ID: 0, StartOffset: 0}

    for i, ins := range insns {
        if leaders[i] && len(cur.Instructions) > 0 {
            // Finish current block
            cur.EndOffset = i * 8 // Each instruction is 8 bytes
            blocks = append(blocks, cur)
            // Start new block
            cur = &BasicBlock{ID: len(blocks), StartOffset: i * 8}
        }
        cur.Instructions = append(cur.Instructions, ins)
    }

    // Add last block
    if len(cur.Instructions) > 0 {
        cur.ID = len(blocks)
        cur.EndOffset = len(insns) * 8
        blocks = append(blocks, cur)
    }

    return blocks
}

// ------------------------------------------------------------
// Build CFG from blocks
// ------------------------------------------------------------

func BuildCFG(blocks []*BasicBlock) *CFG {
    if len(blocks) == 0 {
        return &CFG{Blocks: blocks}
    }

    cfg := &CFG{
        Blocks: blocks,
        Entry:  blocks[0], // First block is entry
        Edges:  []Edge{},
    }

    // Map: instruction offset (byte) → block
    instToBlock := make(map[int]*BasicBlock)
    for _, blk := range blocks {
        // Use instruction offset in bytes
        offset := (blk.StartOffset / 8) * 8 // Approximate, but works for mapping
        instToBlock[offset] = blk
    }

    // Build edges
    for i, blk := range blocks {
        if len(blk.Instructions) == 0 {
            continue
        }

        last := blk.Instructions[len(blk.Instructions)-1]

        // Jump instruction: add jump edge AND fall-through edge
        if last.IsJump() {
            // Calculate jump target (instruction index)
            insnIdx := blk.StartOffset/8 + len(blk.Instructions) - 1
            jumpTargetIdx := insnIdx + 1 + int(last.OffsetVal)
            jumpTargetOffset := jumpTargetIdx * 8

            // Find target block
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

            // Fall-through edge (next block)
            if i+1 < len(blocks) {
                edge := Edge{From: blk, To: blocks[i+1], Type: EdgeFallthrough}
                cfg.Edges = append(cfg.Edges, edge)
                blk.Successors = addUniqueBlock(blk.Successors, blocks[i+1])
                blocks[i+1].Predecessors = addUniqueBlock(blocks[i+1].Predecessors, blk)
            }

        } else if !last.IsExit() {
            // Normal fall-through
            if i+1 < len(blocks) {
                edge := Edge{From: blk, To: blocks[i+1], Type: EdgeFallthrough}
                cfg.Edges = append(cfg.Edges, edge)
                blk.Successors = addUniqueBlock(blk.Successors, blocks[i+1])
                blocks[i+1].Predecessors = addUniqueBlock(blocks[i+1].Predecessors, blk)
            }
        }
    }

    // Detect back-edges (loops)
    cfg.BackEdges = detectBackEdges(cfg)

    return cfg
}

// detectBackEdges uses DFS to find back-edges (loops)
func detectBackEdges(cfg *CFG) []Edge {
    var backEdges []Edge
    visited := make(map[int]bool)
    recStack := make(map[int]bool)

    var dfs func(block *BasicBlock)
    dfs = func(block *BasicBlock) {
        visited[block.ID] = true
        recStack[block.ID] = true

        for _, succ := range block.Successors {
            if recStack[succ.ID] {
                // Found a back-edge
                for _, edge := range cfg.Edges {
                    if edge.From.ID == block.ID && edge.To.ID == succ.ID {
                        edge.Type = EdgeBackEdge
                        backEdges = append(backEdges, edge)
                        break
                    }
                }
            } else if !visited[succ.ID] {
                dfs(succ)
            }
        }

        recStack[block.ID] = false
    }

    if cfg.Entry != nil {
        dfs(cfg.Entry)
    }

    return backEdges
}

// ------------------------------------------------------------
// Detect loops (returns LoopInfo for backward compatibility)
// ------------------------------------------------------------

func DetectLoops(cfg *CFG) []LoopInfo {
    var loops []LoopInfo
    for _, edge := range cfg.BackEdges {
        loops = append(loops, LoopInfo{
            Header: edge.To.ID,
            Blocks: []int{edge.From.ID, edge.To.ID},
        })
    }
    return loops
}
