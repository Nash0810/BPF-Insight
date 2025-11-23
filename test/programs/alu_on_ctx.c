#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

SEC("xdp")
int prog(struct xdp_md *ctx)
{
    long x = (long)ctx;
    x += 5;   // ❌ ctx pointer offset
    return x;
}

char _license[] SEC("license") = "GPL";
