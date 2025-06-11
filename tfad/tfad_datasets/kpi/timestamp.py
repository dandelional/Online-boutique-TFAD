import pandas as pd

def process_timestamp(src_path, des_path):
    # 读取 CSV 文件
    df = pd.read_csv(src_path)

    # 解析 timestamp 列为 datetime 格式
    df['timestamp'] = pd.to_datetime(df['timestamp'], format='%Y/%m/%d %H:%M')

    # 提取年、月、日、小时、分钟等字段
    df['year'] = df['timestamp'].dt.year
    df['month'] = df['timestamp'].dt.month
    df['day'] = df['timestamp'].dt.day
    df['hour'] = df['timestamp'].dt.hour
    df['minute'] = df['timestamp'].dt.minute

    # 组合成一个唯一的数字
    df['datetime_code'] = (df['year'] * 10000000000 +
                        df['month'] * 100000000 +
                        df['day'] * 1000000 +
                        df['hour'] * 10000 +
                        df['minute'] * 100)

    # 删除原始的 timestamp 列（可选）
    df.drop(columns=['timestamp', 'year', 'month', 'day', 'hour', 'minute'], inplace=True)

    # 重命名 datetime_code 列为 timestamp（可选）
    df.rename(columns={'datetime_code': 'timestamp'}, inplace=True)

    # 保存修改后的数据到新的 CSV 文件
    df.to_csv(des_path, index=False)

    print(df.head())

def process_KPI_ID(src_path, des_path):
    df = pd.read_csv(src_path)

    df.rename(columns={'pod': 'KPI ID'}, inplace=True)

    df.to_csv(des_path, index=False)

if __name__ == "__main__":
    process_timestamp('stream_metrics_full_2.csv', 'ob_2.csv')
    process_KPI_ID('ob_2.csv', "ob_2.csv")