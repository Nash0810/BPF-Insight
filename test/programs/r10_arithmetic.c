#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

SEC("xdp")
int prog(struct xdp_md *ctx)
{
    char *x = (char *)(&ctx) + 5;   // ❌ illegal pointer arithmetic
    return x[0];
}

char _license[] SEC("license") = "GPL";
