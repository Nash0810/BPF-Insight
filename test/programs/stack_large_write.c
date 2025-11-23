#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

SEC("xdp")
int prog(struct xdp_md *ctx)
{
    long buf[32];      // 256 bytes
    buf[0] = 123;
    return 0;
}

char _license[] SEC("license") = "GPL";
