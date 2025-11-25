// SPDX-License-Identifier: GPL-2.0
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

// Produces LOTS of sequential instructions.
// Strict profile rule "prog_insn_limit" should fire.

SEC("xdp")
int prog_insn_limit(struct xdp_md *ctx)
{
    volatile __u64 x = 0;

    #pragma clang loop unroll(full)
    for (int i = 0; i < 120; i++) {
        x += i;
    }

    return x ? XDP_PASS : XDP_DROP;
}

char _license[] SEC("license") = "GPL";
