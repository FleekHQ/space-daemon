import { createPow } from '@textile/powergate-client';

// host port is the grpcwebproxy port used to start powergate daemon
const host = 'http://0.0.0.0:6005'; // by default space daemon starts powergate daemon at 6005

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
    });
});
