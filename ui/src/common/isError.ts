export const isError = (output: string): boolean => {
  return output !== "" && !output.startsWith("Unable to find image");
};
