#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

SEC("xdp")
int prog(struct xdp_md *ctx)
{
    int r = bpf_get_prandom_u32();

    if (r)  // ❌ jump depends on unknown scalar
        return 1;

    return 0;
}

char _license[] SEC("license") = "GPL";
