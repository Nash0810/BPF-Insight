// SPDX-License-Identifier: GPL-2.0
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

// Illegal: modifying ctx pointer (r1) via arithmetic.

SEC("xdp")
int ctx_mod(struct xdp_md *ctx)
{
    // ❌ ctx pointer modification — not allowed by verifier
    char *p = (char *)ctx + 4;

    return p != 0;
}

char _license[] SEC("license") = "GPL";
