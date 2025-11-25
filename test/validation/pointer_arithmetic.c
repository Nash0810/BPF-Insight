// SPDX-License-Identifier: GPL-2.0
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

SEC("xdp")
int pointer_arithmetic(struct xdp_md *ctx)
{
    // r1 = ctx, pointer register
    __u64 x = 4;

    // ❌ pointer arithmetic on ctx pointer
    char *ptr = (char *)ctx + x;

    return (int)(long)ptr;
}

char _license[] SEC("license") = "GPL";
