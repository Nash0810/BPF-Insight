#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

SEC("xdp")
int prog(struct xdp_md *ctx)
{
    int x = bpf_get_prandom_u32();

    if (x) bpf_ktime_get_ns();    // helper in branch

    int *ptr = (int *)(ctx->data + x);   // pointer arithmetic

    return *ptr;   // unsafe load
}

char _license[] SEC("license") = "GPL";
