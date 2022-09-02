import Autocomplete from "@mui/material/Autocomplete/Autocomplete";
import TextField from "@mui/material/TextField/TextField";
import { useGetImages } from "../hooks/useGetImages";

interface Props {
  value: string;
  onChange(v: string): void;
}

export const ImageAutocomplete = ({ value, onChange }: Props) => {
  const { data: images } = useGetImages();
  const imageNames = (images || [])
    .reduce((acc, image) => {
      acc = acc.concat(image.RepoTags || []);
      return acc;
    }, [])
    .filter((name) => !name.includes("none"));

  return (
    <Autocomplete
      fullWidth
      disablePortal
      id="image-autocomplete"
      value={value}
      onChange={(_, newValue: string) => onChange(newValue)}
      options={imageNames}
      renderInput={(params) => <TextField {...params} label="Image name" />}
    />
  );
};
