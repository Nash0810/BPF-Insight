#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

SEC("xdp")
int prog(struct xdp_md *ctx)
{
    bpf_ktime_get_ns();
    bpf_ktime_get_ns();
    bpf_get_prandom_u32();
    bpf_get_prandom_u32();
    return 0;
}

char _license[] SEC("license") = "GPL";
