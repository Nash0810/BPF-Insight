#include <linux/bpf.h>
#include <linux/types.h>
#include <bpf/bpf_helpers.h>

SEC("xdp")
int medium_prog(struct xdp_md *ctx) {
    __u64 t = bpf_ktime_get_ns();

    if (t & 1)
        return XDP_DROP;

    if ((t & 2) == 0)
        return XDP_ABORTED;

    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
