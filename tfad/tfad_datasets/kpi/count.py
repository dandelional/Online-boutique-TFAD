import pandas as pd

def calculate_label_ratio(file_path):
    # 读取CSV文件
    df = pd.read_csv(file_path)
    
    # 计算总行数
    total_rows = len(df)
    print(f"total_rows = {total_rows}")
    
    # 计算label=1的行数
    label_1_count = (df['label'] == 1).sum()
    print(f"label_1_count = {label_1_count}")
    
    # 计算比例
    ratio = label_1_count / total_rows
    
    return ratio

# 示例用法
file_path = 'ob_full.csv'  # 替换为你的CSV文件路径
ratio = calculate_label_ratio(file_path)
print(f"Label=1的行占所有行的比例: {ratio:.2%}")



