#include <linux/bpf.h>
#include <linux/types.h>
#include <bpf/bpf_helpers.h>

SEC("xdp")
int helpers_prog(struct xdp_md *ctx) {
    __u64 a = bpf_ktime_get_ns();
    __u64 b = bpf_get_current_pid_tgid();
    __u64 c = bpf_ktime_get_ns();
    volatile __u64 d = bpf_get_prandom_u32();

    if (d & 1)
        return XDP_TX;

    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
