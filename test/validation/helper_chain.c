// SPDX-License-Identifier: GPL-2.0
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

SEC("xdp")
int helper_chain(struct xdp_md *ctx)
{
    __u64 x = bpf_ktime_get_ns();  // helper #1
    __u64 y = bpf_ktime_get_ns();  // helper #2 — same block => warning

    return x + y;
}

char _license[] SEC("license") = "GPL";
