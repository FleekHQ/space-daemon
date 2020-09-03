import { createPow } from '@textile/powergate-client';
import { JobStatus } from "@textile/grpc-powergate-client/dist/ffs/rpc/rpc_pb"

// host port is the grpcwebproxy port used to start powergate daemon
const host = 'http://0.0.0.0:6005'; // by default space daemon starts powergate daemon at 6005

const rawStringToBuffer = ( str ) => {
    let idx, len = str.length, arr = new Array( len );
    for ( idx = 0 ; idx < len ; ++idx ) {
        arr[ idx ] = str.charCodeAt(idx) & 0xFF;
    }
    // You may create an ArrayBuffer from a standard array (of values) as follows:
    return new Uint8Array(arr);
}
describe('Powergate Daemon',  () => {
    const pow = createPow({ host });

    it('should connect and work', async () => {
        const { status } = await pow.health.check();
        expect(status).toEqual(1)
    });

    it('interacting with ffs', async () => {
        // token is the auth token for subsequent calls
        const { token } = await pow.ffs.create();
        pow.setToken(token); // required for subsequent calls
        // get wallet addresses associated with your FFS instance
        const { addrsList } = await pow.ffs.addrs();
        expect(addrsList).toHaveLength(1)
        expect(addrsList[0].name).toEqual('Initial Address')
        console.log('Account Address', addrsList[0].addr);
        console.log('Address Balance', await pow.wallet.balance(addrsList[0].addr));

        // Copied from the CLI... The default config of the js api doesn't work for some reason
        // but the default storage config of CLI works with testnet
        await pow.ffs.setDefaultStorageConfig(
            {
                "hot": {
                    "enabled": true,
                    "allowUnfreeze": false,
                    "ipfs": {
                        "addTimeout": 30
                    }
                },
                "cold": {
                    "enabled": true,
                    "filecoin": {
                        "repFactor": 1,
                        "dealMinDuration": 518400,
                        "excludedMiners": null,
                        "trustedMiners": null,
                        "countryCodes": null,
                        "renew": {
                            "enabled": false,
                            "threshold": 0
                        },
                        "addr": addrsList[0].addr,
                        "maxPrice": 0
                    }
                },
                "repairable": false
            }
        );

        // cache data in IPFS in preparation to store it using FFS
        const { cid } = await pow.ffs.stage(crypto.randomBytes(700));
        // initiates cold storage and deal making
        const { jobId } = await pow.ffs.pushStorageConfig(cid);
        let cancelJobs;
        await new Promise((resolve, reject) => {
            cancelJobs = pow.ffs.watchJobs((job) => {
                console.log('job', job);
                if (job.status === JobStatus.JOB_STATUS_CANCELED) {
                    return reject(new Error('Job Status Cancelled'))
                } else if (job.status === JobStatus.JOB_STATUS_FAILED) {
                    return reject(new Error('Job Status Failed'))
                } else if (job.status === JobStatus.JOB_STATUS_SUCCESS) {
                    return resolve()
                }
            }, jobId);
        });

        if (cancelJobs) {
            cancelJobs();
        }

        const bytes = await pow.ffs.get(cid)
        console.log('retrieved file', bytes);
    }, 500000000);
});