import { FormLabel } from "@material-ui/core";
import FormControl from "@material-ui/core/FormControl";
import FormControlLabel from "@material-ui/core/FormControlLabel";
import Radio from "@material-ui/core/Radio";
import RadioGroup from "@material-ui/core/RadioGroup";
import React from "react";
import { Body2 } from "widgets/Label";

interface KRadioGroupRenderOption {
  value: string;
  label: React.ReactNode;
  explain?: string;
}

interface KFormikRadioGroupRenderProps {
  options: KRadioGroupRenderOption[];
  title?: string;
  error?: boolean;
  name: string;
  value: any;
  onChange: any;
}

export const KFormikRadioGroupRender = ({
  title,
  options,
  error,
  name,
  value,
  onChange,
}: KFormikRadioGroupRenderProps) => {
  return (
    <FormControl component="fieldset" fullWidth margin="dense" error={error}>
      {title ? <FormLabel component="legend">{title}</FormLabel> : null}
      <RadioGroup aria-label={name} name={name} value={value || options[0].value} onChange={onChange}>
        {options.map((option) => {
          return (
            <span key={option.value}>
              <FormControlLabel key={option.value} value={option.value} control={<Radio />} label={option.label} />
              {option.explain ? <Body2 style={{ padding: "0 16px 0 32px" }}>{option.explain}</Body2> : null}
            </span>
          );
        })}
      </RadioGroup>
    </FormControl>
  );
};
