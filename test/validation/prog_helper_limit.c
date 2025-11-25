// SPDX-License-Identifier: GPL-2.0
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

// Calls a helper many times, triggering helper-limit rule.

SEC("xdp")
int prog_helper_limit(struct xdp_md *ctx)
{
    volatile __u64 acc = 0;

    #pragma clang loop unroll(full)
    for (int i = 0; i < 32; i++) {
        acc += bpf_ktime_get_ns();
    }

    return acc ? XDP_PASS : XDP_DROP;
}

char _license[] SEC("license") = "GPL";
