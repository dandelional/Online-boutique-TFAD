import pandas as pd

def reverse_values_in_csv(input_file_path, output_file_path, value_column='value'):
    """
    Args:
        input_file_path (str): 输入CSV文件的路径。
        output_file_path (str): 输出CSV文件的路径。
        value_column (str): 需要反转的列名，默认为'value'。
    """
    # 读取CSV文件
    df = pd.read_csv(input_file_path)

    # 检查指定的列是否存在
    if value_column not in df.columns:
        raise ValueError(f"Column '{value_column}' does not exist in the CSV file.")

    # 反转指定列的值
    df[value_column] = df[value_column].apply(lambda x: 1 - x)

    # 保存到新的CSV文件
    df.to_csv(output_file_path, index=False)
    print(f"Reversed values saved to {output_file_path}")

# 示例用法
if __name__ == "__main__":
    input_csv = "ob_labeled.csv"  # 输入CSV文件路径
    output_csv = "ob_labeled_test.csv"  # 输出CSV文件路径
    value_column = "value"  # 需要反转的列名

    reverse_values_in_csv(input_csv, output_csv, value_column)



