package foundation.icon.btp.bmv.types;

import foundation.icon.btp.bmv.lib.HexConverter;
import foundation.icon.btp.bmv.lib.TypeDecoder;

import java.math.BigInteger;

public class EventDataBTPMessage {

    final static String RLPn = "RLPn";
    private final String next_bmc;
    private final BigInteger seq;
    private final byte[] msg;

    public EventDataBTPMessage(String next_bmc, BigInteger seq, byte[] msg) {
        this.next_bmc = next_bmc;
        this.seq = seq;
        this.msg = msg;
    }


    public static EventDataBTPMessage fromBytes(byte[] indexedValue, byte[] serialized) {
        if (serialized == null)
            return null;
        //TODO: change if the next_bmc is not in indexed value
        //next_bmc
        String next_bmc = HexConverter.bytesToHex(indexedValue);
        TypeDecoder typeDecoder = new TypeDecoder(serialized, 0);

        //seq
        BigInteger seq = TypeDecoder.getUint();
        //String test = TypeDecoder.getString();
        //msg
        byte[] msg = TypeDecoder.getBytes();
        return new EventDataBTPMessage(next_bmc, seq, msg);
    }

    public String getNext_bmc() {
        return next_bmc;
    }

    public BigInteger getSeq() {
        return seq;
    }

    public byte[] getMsg() {
        return msg;
    }
}
