#include <linux/bpf.h>
#include <linux/types.h>
#include <bpf/bpf_helpers.h>

SEC("xdp")
int nested_prog(struct xdp_md *ctx) {
    __u64 t = bpf_ktime_get_ns();

    if (t & 1) {
        if (t & 2) {
            return XDP_DROP;
        } else {
            return XDP_ABORTED;
        }
    }

    if (t & 4) {
        return XDP_TX;
    }

    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
