import os
import json
import torch
from torch import nn

import pytorch_lightning as pl
import tfad
from tfad.ts import TimeSeriesDataset
from tfad.ts import transforms as tr
from tfad.model import TFAD, TFADDataModule

from pytorch_lightning import Trainer
from pytorch_lightning.callbacks import ModelCheckpoint
from pytorch_lightning.loggers import TensorBoardLogger
from torch.utils.data import DataLoader
from pathlib import Path, PosixPath

from typing import Optional, Union
import time

def evaluate_model(ckpt_path, data_dir, output_file):
    """
    Load a pre-trained model and evaluate it on the test set.
    
    :param ckpt_path: Path to the .ckpt file of the trained model.
    :param data_dir: Directory containing the KPI dataset.
    """
    # 1. 加载模型
    model = TFAD.load_from_checkpoint(ckpt_path)
    
    # 2. 加载并预处理测试数据
    _, test_ts_dataset = tfad.datasets.kpi(path=data_dir)  # 假设kpi()函数返回的是(train_set, test_set)
    
    # 标准化（如果训练时有使用scaler）
    scaler = model.scaler if hasattr(model, "scaler") else None  # 如果模型中有scaler就用
    if scaler:
        test_ts_dataset = scaler(test_ts_dataset)
    
    # 创建TimeSeriesDataset对象
    test_dataset = TimeSeriesDataset(test_ts_dataset)
    
    # 3. 创建DataModule
    data_module = TFADDataModule(
        train_ts_dataset=None,
        validation_ts_dataset=None,
        test_ts_dataset=test_dataset,
        window_length=model.hparams.window_length,
        suspect_window_length=model.hparams.suspect_window_length,
        num_series_in_train_batch=4, 
        num_crops_per_series=1,
        label_reduction_method="any",
        stride_val_test=4,  # 可调整
        num_workers=64,
    )
    
    # 4. 创建Trainer
    trainer = pl.Trainer(
        accelerator='auto',
        devices=1,
        logger=False,
    )
    
    # 5. 执行测试
    evaluation_result = trainer.test(model=model, datamodule=data_module)
    
    # 输出测试结果
    print("Test Result:", evaluation_result)

    result_dict = evaluation_result[0]  # 取出第一个字典（通常只有一个结果）

    # 保存为 JSON
    json_path = f"{output_file}.json"
    with open(json_path, "w") as f:
        json.dump(result_dict, f, indent=4)
    print(f"✅ Evaluation results saved to {json_path}")

if __name__ == "__main__":
    CKPT_PATH = "/home/liaotianyin/lishulin/CIKM22-TFAD/tfad/output/kpi/tfad-model-kpi-sup-01-epoch=99-val_f1=0.8571.ckpt"
    DATA_DIR = "/home/liaotianyin/lishulin/CIKM22-TFAD/tfad/tfad_datasets/kpi"
    OUTPUT_PATH = "/home/liaotianyin/lishulin/CIKM22-TFAD/tfad/output/kpi/eval_res_ob"

    evaluate_model(ckpt_path=CKPT_PATH, data_dir=DATA_DIR, output_file=OUTPUT_PATH)