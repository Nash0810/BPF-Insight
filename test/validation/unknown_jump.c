// SPDX-License-Identifier: GPL-2.0
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

// This program introduces an uninitialized register used in a conditional jump.

SEC("xdp")
int unknown_jump(struct xdp_md *ctx)
{
    __u64 x;   // ❌ uninitialized variable
    if (x > 0) // → unknown jump on unbounded value
        return XDP_DROP;

    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
