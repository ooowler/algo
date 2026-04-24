import os
import shutil
import statistics
import time
import urllib.request
from pathlib import Path

import faiss
import h5py
import numpy as np

import plots

ROOT = Path(__file__).resolve().parent
DATA = ROOT / "data" / "sift-128-euclidean.hdf5"
FIG = ROOT / "figures"
RES = ROOT / "results"
URL = "https://ann-benchmarks.com/sift-128-euclidean.hdf5"
K = 100
SIGMA = 3
BENCH_COUNT = int(os.environ.get("BENCH_COUNT", "5"))
TRAIN_SAMPLE = int(os.environ.get("LAB3_TRAIN_SAMPLE", "200000"))


def data_is_valid():
    if not DATA.is_file():
        return False
    try:
        with h5py.File(DATA, "r") as f:
            train = f["train"]
            test = f["test"]
            neighbors = f["neighbors"]
            return train.ndim == 2 and test.ndim == 2 and neighbors.ndim == 2 and test.shape[0] >= 10_000 and neighbors.shape[1] >= K
    except (OSError, KeyError, ValueError):
        return False


def download_data():
    tmp = DATA.with_suffix(DATA.suffix + ".part")
    if tmp.exists():
        tmp.unlink()
    print("download", URL)
    req = urllib.request.Request(URL, headers={"User-Agent": "Mozilla/5.0 (compatible; lab3/1.0)"})
    try:
        with urllib.request.urlopen(req, timeout=600) as r, open(tmp, "wb") as f:
            shutil.copyfileobj(r, f)
        tmp.replace(DATA)
    finally:
        if tmp.exists():
            tmp.unlink()


def ensure_data():
    DATA.parent.mkdir(parents=True, exist_ok=True)
    if data_is_valid():
        return
    if DATA.exists():
        print("remove broken", DATA)
        DATA.unlink()
    download_data()
    if not data_is_valid():
        raise RuntimeError(f"downloaded {DATA} is not a valid SIFT1M HDF5 file")


def load():
    with h5py.File(DATA, "r") as f:
        xb = np.ascontiguousarray(f["train"][:], dtype=np.float32)
        xq = np.ascontiguousarray(f["test"][:], dtype=np.float32)
        gt = np.ascontiguousarray(f["neighbors"][:, :K], dtype=np.int64)
    assert xq.shape[0] >= 10_000
    return xb, xq, gt


def load_demo():
    rng = np.random.default_rng(0)
    nbase, nq, d = 100_000, 10_000, 128
    xb = rng.standard_normal((nbase, d)).astype(np.float32)
    xq = rng.standard_normal((nq, d)).astype(np.float32)
    idx = faiss.IndexFlatL2(d)
    idx.add(xb)
    _, gt = idx.search(xq, K)
    return xb, xq, np.ascontiguousarray(gt, dtype=np.int64)


def index_bytes(idx):
    return int(len(faiss.serialize_index(idx)))


def mean_recall(I, gt):
    nq = I.shape[0]
    t = 0
    for i in range(nq):
        t += np.intersect1d(I[i], gt[i], assume_unique=True).size
    return t / (nq * K)


def sample_training_vectors(xb, minimum):
    if xb.shape[0] <= minimum:
        return xb
    rng = np.random.default_rng(0)
    idx = rng.choice(xb.shape[0], size=minimum, replace=False)
    idx.sort()
    return np.ascontiguousarray(xb[idx], dtype=np.float32)


def search_ms_stats(idx, xq, k, n):
    ms = []
    last_I = None
    nq = float(xq.shape[0])
    for _ in range(max(1, n)):
        t0 = time.perf_counter()
        _, last_I = idx.search(xq, k)
        ms.append((time.perf_counter() - t0) / nq * 1000.0)
    m = statistics.mean(ms)
    s = statistics.stdev(ms) if len(ms) > 1 else 0.0
    return m, s, last_I


def run_lsh(d, xb, xq, gt):
    out = []
    for nbits in (32, 64, 96, 128, 192, 256):
        idx = faiss.IndexLSH(d, nbits)
        t0 = time.perf_counter()
        idx.add(xb)
        t_build = time.perf_counter() - t0
        ms_mean, ms_stdev, I = search_ms_stats(idx, xq, K, BENCH_COUNT)
        out.append(
            {
                "name": f"LSH nbits={nbits}",
                "recall": mean_recall(I, gt),
                "build_s": t_build,
                "search_ms_per_q": ms_mean,
                "search_ms_stdev": ms_stdev,
                "index_bytes": index_bytes(idx),
            }
        )
    return out


def run_hnsw(d, xb, xq, gt):
    out = []
    for M in (16, 32):
        for efc in (40, 200):
            idx = faiss.IndexHNSWFlat(d, M)
            idx.hnsw.efConstruction = efc
            t0 = time.perf_counter()
            idx.add(xb)
            t_build = time.perf_counter() - t0
            for efs in (32, 64, 128, 256):
                idx.hnsw.efSearch = efs
                ms_mean, ms_stdev, I = search_ms_stats(idx, xq, K, BENCH_COUNT)
                out.append(
                    {
                        "name": f"HNSW M={M} efc={efc} efs={efs}",
                        "recall": mean_recall(I, gt),
                        "build_s": t_build,
                        "search_ms_per_q": ms_mean,
                        "search_ms_stdev": ms_stdev,
                        "index_bytes": index_bytes(idx),
                    }
                )
    return out


def run_ivfpq(d, xb, xq, gt):
    out = []
    for nlist in (256, 1024, 4096):
        train_size = max(TRAIN_SAMPLE, 64 * nlist)
        xt = sample_training_vectors(xb, train_size)
        if xt.shape[0] < 64 * nlist:
            continue
        for m in (8, 16, 32):
            if d % m != 0:
                continue
            quantizer = faiss.IndexFlatL2(d)
            idx = faiss.IndexIVFPQ(quantizer, d, nlist, m, 8)
            t0 = time.perf_counter()
            idx.train(xt)
            idx.add(xb)
            t_build = time.perf_counter() - t0
            for nprobe in (1, 4, 8, 16, 32, 64):
                if nprobe > nlist:
                    continue
                idx.nprobe = nprobe
                ms_mean, ms_stdev, I = search_ms_stats(idx, xq, K, BENCH_COUNT)
                out.append(
                    {
                        "name": f"IVFPQ nlist={nlist} m={m} nprobe={nprobe}",
                        "recall": mean_recall(I, gt),
                        "build_s": t_build,
                        "search_ms_per_q": ms_mean,
                        "search_ms_stdev": ms_stdev,
                        "index_bytes": index_bytes(idx),
                    }
                )
    return out


def main():
    try:
        faiss.omp_set_num_threads(int(os.environ.get("LAB3_THREADS", str(os.cpu_count() or 4))))
    except Exception:
        pass
    demo = os.environ.get("LAB3_DEMO") == "1"
    if demo:
        xb, xq, gt = load_demo()
        title_prefix = "Демо (100k базы, L2, GT=Flat): "
    else:
        ensure_data()
        xb, xq, gt = load()
        title_prefix = "SIFT1M: "
    d = xb.shape[1]
    rows = []
    rows.extend(run_lsh(d, xb, xq, gt))
    rows.extend(run_hnsw(d, xb, xq, gt))
    rows.extend(run_ivfpq(d, xb, xq, gt))
    plots.export_all(rows, demo, title_prefix, BENCH_COUNT, SIGMA)
    print("OK", FIG, RES)


if __name__ == "__main__":
    main()
