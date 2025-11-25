// SPDX-License-Identifier: GPL-2.0
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

// Creates a block with many outgoing branches → branch fanout warning.

SEC("xdp")
int branch_fanout(struct xdp_md *ctx)
{
    __u64 x = ctx->data_end - ctx->data;

    // Many conditional jumps forcing wide fanout
    if (x == 1) return XDP_DROP;
    if (x == 2) return XDP_PASS;
    if (x == 3) return XDP_ABORTED;
    if (x == 4) return XDP_TX;
    if (x == 5) return XDP_REDIRECT;

    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
