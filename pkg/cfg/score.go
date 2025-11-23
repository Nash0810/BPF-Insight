package cfg

import (
    "math"
)

type BlockScore struct {
    BlockID       int
    Score         float64
    Insns         int
    Jumps         int
    Helpers       int
    Successors    []int
    IsLoopHeader  bool
    IsLoopBlock   bool
}

type ProgramScore struct {
    TotalScore float64
    Blocks     []BlockScore
    Prediction string
}

var helperCost = map[int]int{
    5:  2, // ktime_get_ns
    14: 1, // get_pid_tgid
    7:  3, // prandom
}

const (
    wALU        = 1.0
    wMove       = 0.5
    wLoadStore  = 2.0
    wCall       = 4.0
    wCondJump   = 3.0
    wUncondJump = 2.0
    wExit       = 0.5

    wLoopHeader = 10.0
    wLoopBody   = 2.0

    wMultiSucc  = 3.0
    wDense      = 1.5
)

func ScoreProgram(cfg *CFG, loops []LoopInfo) ProgramScore {
    loopHeaders := map[int]bool{}
    loopBlocks := map[int]bool{}

    for _, l := range loops {
        loopHeaders[l.Header] = true
        for _, b := range l.Blocks {
            loopBlocks[b] = true
        }
    }

    var scores []BlockScore
    var total float64

    for _, blk := range cfg.Blocks {

        bs := BlockScore{
            BlockID:      blk.ID,
            Insns:        len(blk.Instructions),
            Successors:   blk.Successors,
            IsLoopHeader: loopHeaders[blk.ID],
            IsLoopBlock:  loopBlocks[blk.ID],
        }

        var s float64

        for _, ins := range blk.Instructions {

            // ----------------------------
            // Helper calls
            // ----------------------------
            if ins.IsHelperCall() {
                s += wCall
                s += float64(helperCost[int(ins.Imm)])
                bs.Helpers++
                continue
            }

            // ----------------------------
            // Jumps
            // ----------------------------
            if ins.IsJump() {
                if ins.Opcode == 0x05 {
                    s += wUncondJump
                } else {
                    s += wCondJump
                }
                bs.Jumps++
                continue
            }

            // ----------------------------
            // Exit
            // ----------------------------
            if ins.IsExit() {
                s += wExit
                continue
            }

            // ----------------------------
            // MOV (0xb0 - 0xbf)
            // ----------------------------
            if (ins.Opcode & 0xf0) == 0xb0 {
                s += wMove
                continue
            }

            // ----------------------------
            // Load/store class:
            // 0x00 (ld), 0x01 (ldx), 0x02/03 (st/stx)
            // ----------------------------
            class := ins.Opcode & 0x07
            if class == 0x00 || class == 0x01 || class == 0x02 || class == 0x03 {
                s += wLoadStore
                continue
            }

            // ----------------------------
            // Everything else = ALU
            // ----------------------------
            s += wALU
        }

        // Loop penalties
        if bs.IsLoopHeader {
            s += wLoopHeader
        }
        if bs.IsLoopBlock && !bs.IsLoopHeader {
            s += wLoopBody
        }

        // Multi-successor block penalty
        if len(bs.Successors) > 1 {
            s += wMultiSucc
        }

        // Density penalty
        density := float64(bs.Insns) / float64(len(bs.Successors)+1)
        s += density * wDense

        bs.Score = math.Round(s*100) / 100
        total += bs.Score
        scores = append(scores, bs)
    }

    prediction := "LIKELY PASS"
    if total > 80 {
        prediction = "LIKELY FAIL"
    } else if total > 50 {
        prediction = "BORDERLINE"
    }

    return ProgramScore{
        TotalScore: math.Round(total*100) / 100,
        Blocks:     scores,
        Prediction: prediction,
    }
}
