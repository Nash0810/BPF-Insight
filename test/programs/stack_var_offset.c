#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

SEC("xdp")
int prog(struct xdp_md *ctx)
{
    int x = 10;
    int idx = bpf_get_prandom_u32() & 7;

    int *ptr = (int *)(ctx->data + idx);  // ❌ pointer arithmetic
    return *ptr;
}

char _license[] SEC("license") = "GPL";
