# Copyright 2021 Amazon.com, Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License").
# You may not use this file except in compliance with the License.
# A copy of the License is located at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

import os
import sys
from pathlib import Path, PosixPath
import argparse

import json

import numpy as np

import torch
# from pytorch_lightning.utilities.parsing import str_to_bool

import tfad

def str_to_bool(val):
    if isinstance(val, bool):
        return val
    if isinstance(val, str):
        return val.lower() in ("yes", "true", "t", "1")
    raise ValueError(f"Invalid value for boolean: {val}")

def parse_arguments():

    parser = argparse.ArgumentParser()

    parser.add_argument("--tfad_dir", type=PosixPath)
    parser.add_argument("--data_dir", type=PosixPath)
    parser.add_argument("--hparams_dir", type=PosixPath)
    parser.add_argument("--out_dir", type=PosixPath)
    parser.add_argument("--download_data", type=str_to_bool, default=False)
    parser.add_argument("--yahoo_path", type=PosixPath, default="~")
    parser.add_argument("--number_of_trials", type=int, default=10)
    parser.add_argument("--run_swat", type=str_to_bool, default=True)
    parser.add_argument("--run_yahoo", type=str_to_bool, default=True)
    args, _ = parser.parse_known_args()
    # args = parser.parse_args()

    return args


def main(
    tfad_dir,
    data_dir,
    hparams_dir,
    out_dir,
    download_data,
    yahoo_path=None,
    number_of_trials=10,
    run_swat=True,
    run_yahoo=True,
):


    tfad_dir = tfad_dir.expanduser()
    data_dir = data_dir.expanduser()
    hparams_dir = hparams_dir.expanduser()
    out_dir = out_dir.expanduser()

    sys.path.append(str(tfad_dir / "examples"))

    # Benchmark datasets to consider
    # benchmarks = ["nasa", "smd", "swat", "yahoo"]
    # benchmarks = ["swat", "yahoo", "nasa", "smd", "kpi"]
    benchmarks = ["swat", "yahoo", "kpi"]
    if not run_swat:
        benchmarks.remove("swat")
    if not run_yahoo:
        benchmarks.remove("yahoo")

    # Import pipelines
    for bmk in benchmarks:
        exec(f"from {bmk} import {bmk}_pipeline")

    if download_data:
        yahoo_path = yahoo_path.expanduser()
        try:
            print("Starting data download...")
            print(f"Benchmarks selected for download: {benchmarks}")
            tfad.datasets.download(
                data_dir=data_dir,
                benchmarks=benchmarks,
                yahoo_path=yahoo_path,
            )
            print("Data download completed.")
        except Exception as e:
            print(f"Data download failed: {e}")
            
        # tfad.datasets.download(
        #     data_dir=data_dir,
        #     benchmarks=benchmarks,
        #     yahoo_path=yahoo_path,
        # )

    # Hyperparameter configurations
    hparams_files = [file for file in os.listdir(hparams_dir) if (file.endswith(".json"))]
    hparams_files.sort()
    # Keep only some hyperparameters
    # hparams_files = hparams_files[:2]
    # hparams_files = ['kpi-unsup-01.json']

    for file in hparams_files:
        # file=hparams_files[-1]
        # if not any([file.startswith(bmk) for bmk in benchmarks]):
        if not any([bmk in file for bmk in benchmarks]):
            continue

        print(f"\n Executing hparams: \n {file} \n")
        with open(hparams_dir / file, "r") as f:
            hparams = json.load(f)

        for trail_i in range(number_of_trials):
            # Identify corresponding benckmark dataset
            # bmk = benchmarks[np.where([file.startswith(bmk) for bmk in benchmarks])[0][0]]
            bmk = benchmarks[np.where([bmk in file for bmk in benchmarks])[0][0]]

            # Modify hyperparameters
            hparams.update(
                dict(
                    data_dir=data_dir / bmk,
                    model_dir=out_dir / bmk,
                    log_dir=out_dir / bmk,
                    gpus=1 if torch.cuda.is_available() else 0,
                    evaluation_result_path=out_dir / bmk / file.replace(".json", "_results"+str(trail_i)+".json"),
                    # epochs = 1,
                )
            )

            eval(f"{bmk}_pipeline")(**hparams)


if __name__ == "__main__":

    args = parse_arguments()
    args_dict = vars(args)  # arguments as dictionary
    main(**args_dict)
