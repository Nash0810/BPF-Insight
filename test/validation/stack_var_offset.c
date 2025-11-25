// SPDX-License-Identifier: GPL-2.0
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

SEC("xdp")
int stack_var_offset(struct xdp_md *ctx)
{
    char buf[64];
    __u32 idx = ctx->data & 7;   // dynamic index

    // ❌ verifier: variable stack offset
    buf[idx] = 1;

    return buf[0];
}

char _license[] SEC("license") = "GPL";
