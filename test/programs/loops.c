#include <linux/bpf.h>
#include <linux/types.h>
#include <bpf/bpf_helpers.h>

SEC("xdp")
int loop_prog(struct xdp_md *ctx) {
    int i = 0;
#pragma clang loop unroll(disable)
    while (i < 5) {
        i++;
    }
    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
