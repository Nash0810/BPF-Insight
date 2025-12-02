#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

/* High-complexity eBPF program for testing verifier rejection.
 * 
 * This program intentionally combines multiple complexity factors:
 * - Multiple nested loops with state-dependent loop counts
 * - Extensive conditional branching
 * - Complex pointer arithmetic
 * - High register pressure
 */

SEC("xdp")
int high_complexity_prog(struct xdp_md *ctx) {
    void *data_end = (void *)(long)ctx->data_end;
    void *data = (void *)(long)ctx->data;
    
    #pragma unroll(1)
    for (int i = 0; i < 10; i++) {
        #pragma unroll(1)
        for (int j = 0; j < 10; j++) {
            #pragma unroll(1)
            for (int k = 0; k < 10; k++) {
                if (i + j + k > 20) {
                    if (j * k < 50) {
                        if ((i ^ j) & k > 0) {
                            if (i | j | k == 0xFF) {
                                asm volatile("": : :"memory");
                            }
                        }
                    }
                }
            }
        }
    }
    
    return XDP_DROP;
}

char _license[] SEC("license") = "GPL";
