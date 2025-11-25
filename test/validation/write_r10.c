#include <linux/bpf.h>
#include <linux/types.h>
#include <bpf/bpf_helpers.h>

SEC("xdp")
int write_r10_prog(struct xdp_md *ctx)
{
    volatile int x = 1;

    // Force compiler to emit ALU op on r10:
    asm volatile(
        "r2 = r10\n"     // ok
        "r2 += 4\n"      // ok
        "r10 += 0\n"     // ❌ this attempts to write r10
        :
        :
        :
    );

    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
