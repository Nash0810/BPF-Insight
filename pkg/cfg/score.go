package cfg

import (
	"fmt"
	"math"
	"sort"

	"github.com/cilium/ebpf/asm"
)

// CFGMetrics holds the results of the CFG analysis needed for the total score calculation.
type CFGMetrics struct {
	MaxDepth     int
	AvgBranching float64
}

// BlockComplexity holds the per-block score used for hotspot ranking.
type BlockComplexity struct {
	Block           *BasicBlock
	OffsetRange     string // "start-end" (in BPF instruction number)
	InsnCount       int
	SuccessorCount  int
	IsLoopHeader    bool
	Depth           int
	ComplexityScore float64
	Reason          string
}

// CalculateScores runs all CFG-based analyses: MaxDepth, AvgBranching, and per-block Hotspots.
func CalculateScores(progCFG *CFG, insts []asm.Instruction) (CFGMetrics, []BlockComplexity) {
	// 1. Calculate Per-Block Depth and Branching
	depths := calculateDepths(progCFG)

	totalSuccessors := 0
	for _, block := range progCFG.Blocks {
		totalSuccessors += len(block.Successors)
	}

	maxDepth := 0
	for _, depth := range depths {
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	
	avgBranching := 0.0
	if len(progCFG.Blocks) > 0 {
		avgBranching = float64(totalSuccessors) / float64(len(progCFG.Blocks))
	}


	metrics := CFGMetrics{
		MaxDepth: maxDepth,
		AvgBranching: math.Round(avgBranching*10)/10, // Round for cleaner output
	}

	// 2. Calculate Hotspots (Per-Block Score)
	hotspots := calculateHotspots(progCFG, depths)

	return metrics, hotspots
}

// calculateDepths runs a BFS from the entry block to determine the distance (depth) of each block.
func calculateDepths(progCFG *CFG) map[int]int {
	// ... (Implementation of BFS from Complexity.go) ...
    // Note: Use a standard Breadth-First Search (BFS) implementation here to find shortest path to each block
    // to determine MaxDepth. This implementation is standard.
    
    depths := make(map[int]int)
	if progCFG.Entry == nil {
		return depths
	}
	
	queue := []*BasicBlock{progCFG.Entry}
	depths[progCFG.Entry.ID] = 0

	// Use map for visited status
	visited := make(map[int]bool)
	visited[progCFG.Entry.ID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		currentDepth := depths[current.ID]
		
		for _, succ := range current.Successors {
			// If not visited, set depth and add to queue
			if !visited[succ.ID] {
				visited[succ.ID] = true
				depths[succ.ID] = currentDepth + 1
				queue = append(queue, succ)
			} else {
                // If already visited, it means we found a path, but BFS guarantees shortest path
                // However, for MaxDepth calculation, we might need a modified DFS if true longest path is required.
                // For simplicity and common CFG analysis, BFS/shortest path is acceptable.
            }
		}
	}
	return depths
}

// calculateHotspots calculates a per-block complexity score and ranks the top blocks.
// (Section 5.7, C2)
func calculateHotspots(progCFG *CFG, depths map[int]int) []BlockComplexity {
	complexities := make([]BlockComplexity, len(progCFG.Blocks))
	loopHeaderMap := make(map[int]bool)

	for _, edge := range progCFG.BackEdges {
		loopHeaderMap[edge.To.ID] = true
	}

	for i, block := range progCFG.Blocks {
		bc := BlockComplexity{
			Block: block,
			// Offsets are in bytes, divide by 8 for instruction number
			OffsetRange: fmt.Sprintf("%d-%d", block.StartOffset/8, block.EndOffset/8),
			InsnCount: len(block.Instructions),
			SuccessorCount: len(block.Successors),
			IsLoopHeader: loopHeaderMap[block.ID],
			Depth: depths[block.ID],
		}
		
		// Per-Block Complexity Formula (Heuristic)
		score := 0.0
		
		// 1. Instruction count (0.5 point per instruction)
		score += float64(bc.InsnCount) * 0.5
		
		// 2. Branching factor (5.0 points per successor over 1)
		if bc.SuccessorCount > 1 {
			score += float64(bc.SuccessorCount) * 5.0
		}
		
		// 3. Loop contribution (20.0 points for loop headers)
		if bc.IsLoopHeader {
			score += 20.0
			bc.Reason = "Loop header (source of back-edge)"
		}

		// 4. Depth contribution (2.0 points per depth level)
		score += float64(bc.Depth) * 2.0
		if bc.Depth > 8 { 
			bc.Reason = "Deep nesting/control path (depth " + fmt.Sprintf("%d)", bc.Depth)
		} else if bc.Reason == "" {
			bc.Reason = fmt.Sprintf("High instruction count (%d insns)", bc.InsnCount)
		}
		
		bc.ComplexityScore = math.Round(score*10)/10 // Round to 1 decimal
		complexities[i] = bc
	}
	
	// Sort by complexity score descending (Hotspot Ranking)
	sort.Slice(complexities, func(i, j int) bool {
		return complexities[i].ComplexityScore > complexities[j].ComplexityScore
	})
	
	// Return top 10 hotspots (or up to 50 blocks) with a minimum score
	hotspots := make([]BlockComplexity, 0)
	for i, c := range complexities {
		if c.ComplexityScore < 5.0 || i >= 10 {
			break
		}
		hotspots = append(hotspots, c)
	}

	return hotspots
}